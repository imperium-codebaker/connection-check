// Package geo estimates where traced routers sit from the city codes carriers
// put in their reverse-DNS names, checked against round-trip times.
package geo

import (
	"math"
	"sort"
)

const (
	// kmPerMs is how far fiber reaches per millisecond of round trip (about 200 km/ms one way).
	kmPerMs = 100.0
	// slackKm absorbs noisy router reply times when comparing two hops.
	slackKm = 300.0
	// nearUserMs is the round trip below which a hop is taken to be near the user.
	nearUserMs = 5.0
	// nearUserKm is the most that two near-user hops may be apart.
	nearUserKm = 500.0
)

// Hop is one traced hop as seen by the placer.
type Hop struct {
	Name   string  // reverse-DNS name, "" if none
	MinRTT float64 // best round trip in ms, 0 if the hop never answered
}

// Place is an estimated hop location.
//
// Approx is false when the place came from a CLLI city+state code or from an
// airport code with a site number attached (such as "lax01"), and true when it
// came from a lone three-letter token that only happens to match an airport code.
type Place struct {
	City   string
	Lat    float64
	Lon    float64
	Approx bool
}

// placer accepts candidates one at a time, keeping each hop's accepted candidate.
type placer struct {
	hops   []Hop
	placed []*candidate
}

// PlaceRoute takes the hops of one trace in order and returns one entry per
// hop: a Place, or nil when the hop can't be placed.
func PlaceRoute(hops []Hop) []*Place {
	hops = capRTTs(hops)
	p := placer{hops: hops, placed: make([]*candidate, len(hops))}
	for _, c := range rankedCandidates(hops) {
		if p.placed[c.hop] == nil && p.accepts(c) {
			p.placed[c.hop] = &c
		}
	}
	return p.places()
}

// capRTTs returns a copy of hops with each MinRTT capped at the lowest MinRTT of any
// later answering hop: a hop is no farther away than the hops beyond it, and some
// routers answer pings slowly.
func capRTTs(hops []Hop) []Hop {
	out := append([]Hop(nil), hops...)
	low := math.Inf(1)
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].MinRTT <= 0 {
			continue
		}
		low = math.Min(low, out[i].MinRTT)
		out[i].MinRTT = low
	}
	return out
}

// rankedCandidates returns every hop's candidates, strongest first and in hop order within a strength.
func rankedCandidates(hops []Hop) []candidate {
	var all []candidate
	for i, h := range hops {
		all = append(all, candidatesFor(i, h.Name)...)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].strength > all[j].strength })
	return all
}

// accepts reports whether c fits every placement accepted so far.
func (p *placer) accepts(c candidate) bool {
	if c.strength == bareIATA && p.hops[c.hop].MinRTT <= 0 {
		return false
	}
	for _, o := range p.placed {
		if o != nil && !p.consistent(c, *o) {
			return false
		}
	}
	return c.strength > bareIATA || p.corroborated(c)
}

// consistent reports whether two placed hops are close enough for the
// difference in their round trips. Hops that never answered are not checked.
func (p *placer) consistent(c, o candidate) bool {
	early, late := p.hops[o.hop], p.hops[c.hop]
	if c.hop < o.hop {
		early, late = late, early
	}
	if early.MinRTT <= 0 || late.MinRTT <= 0 {
		return true
	}
	d := distanceKm(c.site, o.site)
	if early.MinRTT < nearUserMs && late.MinRTT < nearUserMs && d > nearUserKm {
		return false
	}
	return d <= kmPerMs*math.Max(0, late.MinRTT-early.MinRTT)+slackKm
}

// corroborated reports whether a bare-token candidate lies within slackKm of a
// stronger placement, or whether the route has no stronger placement at all.
func (p *placer) corroborated(c candidate) bool {
	strong := false
	for _, o := range p.placed {
		if o == nil || o.strength == bareIATA {
			continue
		}
		strong = true
		if distanceKm(c.site, o.site) <= slackKm {
			return true
		}
	}
	return !strong
}

// places converts the accepted candidates to Places, nil where a hop has none.
func (p *placer) places() []*Place {
	out := make([]*Place, len(p.placed))
	for i, c := range p.placed {
		if c != nil {
			out[i] = &Place{City: c.site.City, Lat: c.site.Lat, Lon: c.site.Lon, Approx: c.strength == bareIATA}
		}
	}
	return out
}
