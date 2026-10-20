// Package stretch groups a traced route into the stretches the page and the text report
// show: home, provider, handoff carriers, an unseen private link, and the site.
package stretch

import (
	"math"
	"slices"
	"strings"

	"github.com/imperium-codebaker/connection-check/internal/owner"
	"github.com/imperium-codebaker/connection-check/internal/report"
)

// Kinds of stretch, in route order.
const (
	Home     = "home"
	Provider = "provider"
	Handoff  = "handoff"
	Link     = "link"
	Site     = "site"
)

// LinkMs is the least delay between the last answering router and the site that counts
// as an unseen link.
const LinkMs = 20

// NearMs is the largest best round trip at a stretch's end for the stretch to count as near
// the user: about 1,000 km of fibre.
const NearMs = 10

var gradeRank = map[string]int{"good": 1, "fair": 2, "poor": 3}

// group is a run of hops with one kind; idx holds indexes into the hops.
type group struct {
	kind string
	idx  []int
}

// Build returns the stretches of a route: hops before the destination in order, the
// destination, and the verdict for the same window.
func Build(hops []report.Hop, dest report.Hop, v report.Verdict) []report.Stretch {
	reach := reached(hops, dest, func(h report.Hop) float64 { return h.Avg })
	best := reached(hops, dest, func(h report.Hop) float64 { return h.Best })
	groups := split(hops)
	out := make([]report.Stretch, 0, len(groups)+2)
	for _, g := range groups {
		s := routeStretch(g, hops, reach)
		s.Near = g.kind == Home || nearEnd(g, best)
		out = append(out, s)
	}
	site := siteStretch(lastSite(&out), dest)
	site.Near = answers(dest) && dest.Best > 0 && dest.Best <= NearMs
	if link, ok := linkStretch(out, hops, dest, reach); ok {
		link.Name = linkName(out, site)
		out = append(out, link)
	}
	out = append(out, site)
	markBlame(out, v)
	return out
}

// kindOf maps an owner to a stretch kind; "" means the hop takes its neighbour's.
func kindOf(o string) string {
	switch owner.Owner(o) {
	case owner.Home, owner.Provider, owner.Handoff, owner.Site:
		return o
	}
	return ""
}

// split groups runs of hops with the same kind. An unknown hop joins the run before it,
// unless that run is home: hops past the home router belong to the network beyond it.
// Unknown hops that join no run before them wait for the next run, or join the site when
// none follows.
func split(hops []report.Hop) []group {
	var groups []group
	var lead []int
	for i, h := range hops {
		k := kindOf(h.Owner)
		switch {
		case k == "" && (len(groups) == 0 || groups[len(groups)-1].kind == Home):
			lead = append(lead, i)
		case k == "" || len(groups) > 0 && k == groups[len(groups)-1].kind:
			last := &groups[len(groups)-1]
			last.idx = append(append(last.idx, lead...), i)
			lead = nil
		default:
			groups = append(groups, group{kind: k, idx: append(lead, i)})
			lead = nil
		}
	}
	if len(lead) > 0 {
		groups = append(groups, group{kind: Site, idx: lead})
	}
	return groups
}

// reached returns each answering hop's round trip, read by rtt, capped at the lowest of
// every later answering hop and the destination, since a router that answers pings slowly
// is no farther away than the routers beyond it; -1 marks a hop with no figure.
func reached(hops []report.Hop, dest report.Hop, rtt func(report.Hop) float64) []float64 {
	out := make([]float64, len(hops))
	low := math.Inf(1)
	if answers(dest) && rtt(dest) > 0 {
		low = rtt(dest)
	}
	for i := len(hops) - 1; i >= 0; i-- {
		out[i] = -1
		if answers(hops[i]) && rtt(hops[i]) > 0 {
			low = min(low, rtt(hops[i]))
			out[i] = low
		}
	}
	return out
}

// nearEnd reports whether the last answering hop of g has a best round trip within NearMs.
func nearEnd(g group, best []float64) bool {
	for i := len(g.idx) - 1; i >= 0; i-- {
		if b := best[g.idx[i]]; b >= 0 {
			return b <= NearMs
		}
	}
	return false
}

func answers(h report.Hop) bool { return h.Answered && h.Avg > 0 }

func routeStretch(g group, hops []report.Hop, reach []float64) report.Stretch {
	s := report.Stretch{Kind: g.kind, FirstTTL: hops[g.idx[0]].TTL, LastTTL: hops[g.idx[len(g.idx)-1]].TTL,
		Routers: len(g.idx)}
	for _, i := range g.idx {
		h := hops[i]
		if reach[i] >= 0 {
			s.Ms = round1(reach[i])
		}
		if h.Place != nil {
			s.City, s.Approx = h.Place.City, h.Place.Approx
		}
		s.Grade = worse(s.Grade, h)
		if g.kind != Home {
			s.Name = addName(s.Name, owner.ShortName(h.ASName))
		}
	}
	return s
}

// lastSite removes a final run of site-owned hops from out and returns it, or an empty
// site stretch when the route has none.
func lastSite(out *[]report.Stretch) report.Stretch {
	if n := len(*out); n > 0 && (*out)[n-1].Kind == Site {
		s := (*out)[n-1]
		*out = (*out)[:n-1]
		return s
	}
	return report.Stretch{Kind: Site}
}

// siteStretch adds the destination to s, the site-owned hops before it.
func siteStretch(s report.Stretch, dest report.Hop) report.Stretch {
	s.Routers++
	s.Ms = 0
	if answers(dest) {
		s.Ms = round1(dest.Avg)
	}
	if dest.Place != nil {
		s.City, s.Approx = dest.Place.City, dest.Place.Approx
	}
	s.Grade = worse(s.Grade, dest)
	s.Name = owner.ShortName(dest.ASName)
	if s.Name == "" {
		s.Name = dest.Name
	}
	return s
}

// linkStretch returns the unseen stretch between the last answering router and the
// destination, when that router lies before the site's own hops and the gap is at least LinkMs.
func linkStretch(out []report.Stretch, hops []report.Hop, dest report.Hop, reach []float64) (report.Stretch, bool) {
	if !answers(dest) || len(out) == 0 {
		return report.Stretch{}, false
	}
	end := out[len(out)-1].LastTTL
	for i := len(hops) - 1; i >= 0; i-- {
		if reach[i] < 0 {
			continue
		}
		if gap := dest.Avg - reach[i]; hops[i].TTL <= end && gap >= LinkMs {
			return report.Stretch{Kind: Link, Ms: math.Round(gap)}, true
		}
		break
	}
	return report.Stretch{}, false
}

// linkName names a link by the cities at its two ends, or "" when either is unknown.
func linkName(before []report.Stretch, site report.Stretch) string {
	from := ""
	for _, s := range before {
		if s.City != "" {
			from = s.City
		}
	}
	if from == "" || site.City == "" {
		return ""
	}
	return from + " to " + site.City
}

// markBlame flags the stretch where the verdict says the trouble starts.
func markBlame(out []report.Stretch, v report.Verdict) {
	for i := range out {
		s := &out[i]
		switch {
		case v.BlameTTL > 0:
			s.Blamed = s.Kind != Link && s.FirstTTL > 0 && s.FirstTTL <= v.BlameTTL && v.BlameTTL <= s.LastTTL
		default:
			s.Blamed = v.Owner == string(owner.Site) && s.Kind == Site
		}
	}
}

func worse(grade string, h report.Hop) string {
	if gradeRank[h.Grade] > gradeRank[grade] {
		return h.Grade
	}
	return grade
}

func addName(names, name string) string {
	switch {
	case name == "" || slices.Contains(strings.Split(names, ", "), name):
		return names
	case names == "":
		return name
	}
	return names + ", " + name
}

func round1(x float64) float64 { return math.Round(x*10) / 10 }
