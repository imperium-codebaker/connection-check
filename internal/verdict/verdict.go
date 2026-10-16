package verdict

import "github.com/imperium-codebaker/connection-check/internal/owner"

// Hop is one hop's figures over the window. Loss is in percent, times in ms.
type Hop struct {
	TTL      int         `json:"ttl"`
	Owner    owner.Owner `json:"owner"`
	Answered bool        `json:"answered"`
	Sent     int         `json:"sent"`
	Loss     float64     `json:"loss"`
	Avg      float64     `json:"avg"`
	Jitter   float64     `json:"jitter"`
}

func (h Hop) value(m Measure) float64 {
	switch m {
	case Loss:
		return h.Loss
	case Jitter:
		return h.Jitter
	}
	return h.Avg
}

// Input is everything Judge needs.
type Input struct {
	Site     string `json:"site"`     // the site's display name
	Provider string `json:"provider"` // the provider's network name, if known
	Hops     []Hop  `json:"hops"`     // hops before the destination, in route order
	Dest     Hop    `json:"dest"`
}

// Verdict is the judgement for a window.
type Verdict struct {
	Grade    Grade
	Measure  Measure
	Owner    owner.Owner // "" when the problem is spread along the route
	BlameTTL int         // hop where the problem starts; 0 for the destination or no single hop
	LongRoad bool
	Headline string
	Action   string
}

// Judge grades the destination and, when it isn't good, finds where the trouble starts.
// Only the destination decides the grade: trouble at a middle hop that doesn't carry
// through to the destination is a router ignoring pings, not a fault.
func Judge(in Input) Verdict {
	if in.Dest.Sent < MinSamples {
		return Verdict{Grade: Measuring, Headline: "Measuring your connection to " + in.Site + "…",
			Action: "Give it a minute."}
	}
	g, m := GradeHop(in.Dest)
	v := Verdict{Grade: g, Measure: m}
	located := true
	if g != Good {
		v.BlameTTL, v.Owner, located = blame(in, m)
		v.LongRoad = m == Latency && g == Fair && isFar(v.Owner) && GradeOf(Loss, in.Dest.Loss) == Good &&
			GradeOf(Jitter, in.Dest.Jitter) == Good
	}
	v.Headline, v.Action = words(in, v, located)
	return v
}

func isFar(o owner.Owner) bool { return o == owner.Handoff || o == owner.Site }

// blame returns the hop where measure m's trouble starts and who owns it.
// Only trouble that carries through to the destination counts (carriedValues); the biggest
// rise in the carried figure marks the start. A rise under half the destination's figure
// means no single culprit. located is false when the hops got too few pings to tell.
func blame(in Input, m Measure) (ttl int, who owner.Owner, located bool) {
	route := append(answered(in.Hops), in.Dest)
	carried, located := carriedValues(route, m)
	if !located {
		return 0, "", false
	}
	at, rise, prev := -1, 0.0, 0.0
	for i, v := range carried {
		if v-prev > rise {
			at, rise = i, v-prev
		}
		prev = v
	}
	if at < 0 || rise < carried[len(carried)-1]/2 {
		return 0, "", true
	}
	if at == len(route)-1 {
		return 0, owner.Site, true
	}
	return route[at].TTL, route[at].Owner, true
}

func answered(hops []Hop) []Hop {
	var out []Hop
	for _, h := range hops {
		if answering(h) {
			out = append(out, h)
		}
	}
	return out
}

func answering(h Hop) bool { return h.Answered && h.Sent > 0 && h.Loss < 100 }
