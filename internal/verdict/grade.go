// Package verdict grades a connection and names who owns the problem.
package verdict

// Grade is a Good/Fair/Poor rating on HP Anyware's scales, plus a state for too little data.
type Grade string

// Grades, best first.
const (
	Measuring Grade = "measuring"
	Good      Grade = "good"
	Fair      Grade = "fair"
	Poor      Grade = "poor"
)

// Measure names what drove a grade.
type Measure string

// Measures, in the order they win ties.
const (
	Loss    Measure = "loss"
	Jitter  Measure = "jitter"
	Latency Measure = "latency"
)

// MinSamples is the fewest destination probes a verdict needs.
const MinSamples = 30

// Bands: latency from HP's Client Performance Health Check (warn above 100 ms, fail above
// 250 ms); loss from HP's in-session Connection Health indicator; HP's 30 ms variation
// ceiling sets the Poor jitter line.
var bands = map[Measure][2]float64{
	Latency: {100, 250},  // ms: Fair above the first, Poor above the second
	Loss:    {0.25, 0.5}, // percent
	Jitter:  {10, 30},    // ms
}

// GradeOf grades one value of a measure.
func GradeOf(m Measure, v float64) Grade {
	b := bands[m]
	switch {
	case v > b[1]:
		return Poor
	case v > b[0]:
		return Fair
	}
	return Good
}

// GradeHop returns a hop's worst grade across loss, jitter and latency, and the measure behind it.
func GradeHop(h Hop) (Grade, Measure) {
	worst, by := Good, Measure("")
	for _, m := range []Measure{Loss, Jitter, Latency} {
		if g := GradeOf(m, h.value(m)); rank(g) > rank(worst) {
			worst, by = g, m
		}
	}
	return worst, by
}

// HopGrades grades each hop on only the trouble that carries through to the destination
// (carriedValues), so a router that merely ignores or delays pings isn't shown as a fault.
// Unanswered hops get "".
func HopGrades(hops []Hop, dest Hop) []Grade {
	route := append(answered(hops), dest)
	loss, _ := carriedValues(route, Loss)
	avg, _ := carriedValues(route, Latency)
	jitter, _ := carriedValues(route, Jitter)
	out := make([]Grade, len(hops))
	j := 0
	for i, h := range hops {
		if answering(h) {
			out[i], _ = GradeHop(Hop{Loss: loss[j], Avg: avg[j], Jitter: jitter[j]})
			j++
		}
	}
	return out
}

// IgnoresPings reports, for each hop, whether it is a router rationing its own replies: it
// answers, loses more than the Poor share of pings, and no later hop loses more than that
// share, so its loss never reaches the destination.
func IgnoresPings(hops []Hop, dest Hop) []bool {
	poor := bands[Loss][1]
	out := make([]bool, len(hops))
	low := 100.0
	if answering(dest) {
		low = dest.Loss
	}
	for i := len(hops) - 1; i >= 0; i-- {
		if h := hops[i]; answering(h) {
			out[i] = h.Loss > poor && low <= poor
			low = min(low, h.Loss)
		}
	}
	return out
}

func rank(g Grade) int {
	switch g {
	case Fair:
		return 1
	case Poor:
		return 2
	}
	return 0
}
