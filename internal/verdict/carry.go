package verdict

import (
	"math"
	"slices"

	"github.com/johncbrubaker/connection-check/internal/owner"
)

// Loss-start model. A hop's reply loss is either the loss it carries (the destination's
// rate from the start onward, quietLoss before it) or, for a share of routers that ration
// their own replies, any rate at all.
const (
	ownLossShare = 0.25    // prior share of routers whose own reply loss hides what they carry
	quietLoss    = 0.001   // loss rate of a hop that carries no trouble
	sureShare    = 2.0 / 3 // weight one owner's start hops need before the start is named: twice all others
)

// carriedValues returns each hop's figure for m counting only trouble that carries through
// to the destination; route is the answering hops with the destination last. Each figure is
// capped at the lowest figure of every later hop. For loss, the hops from lossStart onward
// first take the destination's loss, so a hop that lost fewer pings by chance doesn't hide
// the loss before it. located is false when the hops got too few pings to place the start.
func carriedValues(route []Hop, m Measure) (carried []float64, located bool) {
	last := len(route) - 1
	start, located := last, true
	if m == Loss {
		if start, located = lossStart(route); !located {
			start = last
		}
	}
	carried = make([]float64, len(route))
	low := route[last].value(m)
	for i := last; i >= 0; i-- {
		if i < start {
			low = min(low, route[i].value(m))
		}
		carried[i] = low
	}
	return carried, located
}

// lossStart returns the hop where the destination's loss most likely starts, scoring every
// possible start against the whole route at once, as PingPlotter's guide does by eye when
// it looks for the first hop that shows the destination's problem. ok is false when no
// owner's start hops hold sureShare of the weight, meaning too few pings to tell.
func lossStart(route []Hop) (start int, ok bool) {
	last := len(route) - 1
	p := min(route[last].Loss/100, 0.999)
	score := make([]float64, len(route)) // log-likelihood of starting at s, against starting at the destination
	start = last
	for s := last - 1; s >= 0; s-- {
		score[s] = score[s+1] + carryEvidence(route[s], p)
		if score[s] > score[start] {
			start = s
		}
	}
	return start, ownerShare(route, score, startOwner(route, start)) >= sureShare
}

// carryEvidence is the log-likelihood ratio that hop h carries loss rate p rather than none.
func carryEvidence(h Hop, p float64) float64 {
	n := float64(h.Sent)
	k := math.Round(n * h.Loss / 100)
	return replyLoss(k, n, p) - replyLoss(k, n, quietLoss)
}

// replyLoss is the log-probability of a hop losing k of n replies when it carries rate r:
// binomial at r, or, for a router rationing its own replies, uniform over every count.
func replyLoss(k, n, r float64) float64 {
	lg := func(x float64) float64 { v, _ := math.Lgamma(x); return v }
	binomial := lg(n+1) - lg(k+1) - lg(n-k+1) + k*math.Log(r) + (n-k)*math.Log(1-r)
	return logAdd(math.Log(1-ownLossShare)+binomial, math.Log(ownLossShare)-math.Log(n+1))
}

// ownerShare returns the weight of the starts owned by who, with each start s weighted by
// exp(score[s]).
func ownerShare(route []Hop, score []float64, who owner.Owner) float64 {
	top := slices.Max(score)
	mine, all := 0.0, 0.0
	for s, v := range score {
		w := math.Exp(v - top)
		all += w
		if startOwner(route, s) == who {
			mine += w
		}
	}
	return mine / all
}

// startOwner names who owns trouble that starts at route[s]; the destination is the site.
func startOwner(route []Hop, s int) owner.Owner {
	if s == len(route)-1 {
		return owner.Site
	}
	return route[s].Owner
}

func logAdd(a, b float64) float64 {
	hi, lo := max(a, b), min(a, b)
	return hi + math.Log1p(math.Exp(lo-hi))
}
