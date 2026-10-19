package monitor

import (
	"math"
	"time"

	"github.com/imperium-codebaker/connection-check/internal/geo"
	"github.com/imperium-codebaker/connection-check/internal/lookup"
	"github.com/imperium-codebaker/connection-check/internal/owner"
	"github.com/imperium-codebaker/connection-check/internal/report"
	"github.com/imperium-codebaker/connection-check/internal/stats"
	"github.com/imperium-codebaker/connection-check/internal/verdict"
)

// snapshot assembles what the page shows for site s over window w at time now.
func (m *Monitor) snapshot(s *site, w window, now time.Time) report.Snapshot {
	from, to := w.span(now)
	sums := m.hopSummaries(s, from, to)
	infos := m.infos(sums)
	destInfo := m.look.Get(s.addr)
	owners := owner.Classify(ownerHops(sums, infos), destInfo.ASN)
	dest := s.route.Dest(from, to)
	n := len(sums)
	places := geo.PlaceRoute(geoHops(append(sums[:n:n], dest), append(infos[:n:n], destInfo)))
	if p := geo.Airport(s.cfg.Place); p != nil {
		places[n] = p
	}
	hops := make([]report.Hop, n)
	for i := range sums {
		hops[i] = hopRow(i+1, sums[i], infos[i], owners[i], places[i])
	}
	d := hopRow(0, dest, destInfo, owner.Site, places[n])
	d.Addr = s.addr.String()
	gradeHops(hops, &d)
	return report.Snapshot{
		V: 1, Site: s.cfg.Name, Host: s.cfg.Host, Addr: s.addr.String(),
		From: from.UnixMilli(), To: to.UnixMilli(), OS: goos, Mode: m.modeInfo(), Window: w.kind,
		Verdict: judge(s.cfg.Name, hops, d), Hops: hops, Dest: d, Timeline: timeline(s.route, now),
	}
}

// hopSummaries summarises the hops before the destination, dropping trailing hops that never answered.
func (m *Monitor) hopSummaries(s *site, from, to time.Time) []stats.Summary {
	sums := s.route.Hops(m.eng.PathLen(s.addr)-1, from, to)
	for len(sums) > 0 && !answered(sums[len(sums)-1]) {
		sums = sums[:len(sums)-1]
	}
	return sums
}

func (m *Monitor) infos(sums []stats.Summary) []lookup.Info {
	out := make([]lookup.Info, len(sums))
	for i, h := range sums {
		if h.From.IsValid() {
			out[i] = m.look.Get(h.From)
		}
	}
	return out
}

func answered(s stats.Summary) bool { return s.Sent > s.Lost }

func ownerHops(sums []stats.Summary, infos []lookup.Info) []owner.Hop {
	out := make([]owner.Hop, len(sums))
	for i, h := range sums {
		out[i] = owner.Hop{Addr: h.From, ASN: infos[i].ASN}
	}
	return out
}

func geoHops(sums []stats.Summary, infos []lookup.Info) []geo.Hop {
	out := make([]geo.Hop, len(sums))
	for i, h := range sums {
		if answered(h) {
			out[i] = geo.Hop{Name: infos[i].Name, MinRTT: h.Best}
		}
	}
	return out
}

func hopRow(ttl int, s stats.Summary, info lookup.Info, o owner.Owner, p *geo.Place) report.Hop {
	h := report.Hop{
		TTL: ttl, Name: info.Name, ASN: info.ASN, ASName: info.ASName, Owner: string(o),
		Answered: answered(s), Sent: s.Sent, Lost: s.Lost, Loss: round(s.Loss, 2),
		Avg: round(s.Avg, 1), Best: round(s.Best, 1), Worst: round(s.Worst, 1), Jitter: round(s.Jitter, 1),
	}
	if s.From.IsValid() {
		h.Addr = s.From.String()
	}
	if p != nil {
		h.Place = &report.Place{City: p.City, Lat: p.Lat, Lon: p.Lon, Approx: p.Approx, Given: p.Given}
	}
	return h
}

// gradeHops sets each row's grade; middle hops are graded only on trouble that reaches the destination.
func gradeHops(hops []report.Hop, dest *report.Hop) {
	vh := make([]verdict.Hop, len(hops))
	for i, h := range hops {
		vh[i] = verdictHop(h)
	}
	for i, g := range verdict.HopGrades(vh, verdictHop(*dest)) {
		hops[i].Grade = string(g)
	}
	if dest.Answered {
		g, _ := verdict.GradeHop(verdictHop(*dest))
		dest.Grade = string(g)
	}
}

func judge(site string, hops []report.Hop, dest report.Hop) report.Verdict {
	in := verdict.Input{Site: site, Dest: verdictHop(dest)}
	for _, h := range hops {
		in.Hops = append(in.Hops, verdictHop(h))
		if in.Provider == "" && h.Owner == string(owner.Provider) {
			in.Provider = h.ASName
		}
	}
	v := verdict.Judge(in)
	return report.Verdict{
		Grade: string(v.Grade), Measure: string(v.Measure), Owner: string(v.Owner),
		BlameTTL: v.BlameTTL, LongRoad: v.LongRoad, Headline: v.Headline, Action: v.Action,
	}
}

func verdictHop(h report.Hop) verdict.Hop {
	return verdict.Hop{TTL: h.TTL, Owner: owner.Owner(h.Owner), Answered: h.Answered, Sent: h.Sent,
		Loss: h.Loss, Avg: h.Avg, Jitter: h.Jitter}
}

func timeline(r *stats.Route, now time.Time) []report.Bucket {
	end := now.Truncate(bucketWidth)
	out := []report.Bucket{}
	for _, b := range r.Timeline(end.Add(-timelineSpan), end, bucketWidth) {
		out = append(out, report.Bucket{T: b.Start.UnixMilli(), Sent: b.Sent, Lost: b.Lost, Avg: round(b.Avg, 1)})
	}
	return out
}

func round(x float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(x*p) / p
}
