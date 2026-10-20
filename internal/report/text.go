package report

import (
	"fmt"
	"strings"
	"time"
)

var windowNames = map[string]string{
	"default": "last 5 minutes",
	"minute":  "last minute",
	"all":     "whole chart",
	"bad":     "marked as a bad moment",
	"custom":  "selected range",
}

// KindLabels name each kind of stretch.
var KindLabels = map[string]string{
	"home": "Home", "provider": "Provider", "handoff": "Handoff", "link": "Private link", "site": "Site",
}

// Text returns a plain-text summary of s for email or chat.
func Text(s Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Connection Check report: %s (%s, %s)\n", s.Site, s.Host, s.Addr)
	fmt.Fprintf(&b, "Window: %s (%s), %s, %s probes\n", span(s.From, s.To), windowNames[s.Window], s.OS, s.Mode.Name)
	fmt.Fprintf(&b, "Verdict: %s. %s\n%s\n\n", strings.ToUpper(s.Verdict.Grade), s.Verdict.Headline, s.Verdict.Action)
	if len(s.Stretches) > 0 {
		writeStretches(&b, s.Stretches)
	}
	fmt.Fprintf(&b, "%3s  %-46s %-14s %6s %7s %7s\n", "#", "Router", "Where", "Loss", "Avg ms", "Jitter")
	writeHops(&b, s)
	writeChanges(&b, s.RouteChanges)
	return b.String()
}

func writeStretches(b *strings.Builder, list []Stretch) {
	b.WriteString("Route:\n")
	for i, st := range list {
		what := strings.Join(nonEmpty(st.Name, st.City), ", ")
		figure := fmt.Sprintf("%.1f ms, %d routers, %s", st.Ms, st.Routers, st.Grade)
		if st.Kind == "link" {
			figure = fmt.Sprintf("+%.0f ms, no routers show", st.Ms)
		}
		blame := ""
		if st.Blamed {
			blame = "  <- trouble starts here"
		}
		fmt.Fprintf(b, "%3d  %-12s %-36s %s%s\n", i+1, KindLabels[st.Kind], what, figure, blame)
	}
	b.WriteString("\n")
}

// writeHops writes the hop rows, under a heading for each stretch when the snapshot has them.
func writeHops(b *strings.Builder, s Snapshot) {
	if len(s.Stretches) == 0 {
		for _, h := range s.Hops {
			b.WriteString(row(fmt.Sprint(h.TTL), h))
		}
		b.WriteString(row("end", s.Dest))
		return
	}
	for _, st := range s.Stretches {
		fmt.Fprintf(b, "  %s\n", strings.Join(nonEmpty(KindLabels[st.Kind], st.Name), ": "))
		if st.Kind == "link" {
			fmt.Fprintf(b, "%3s  %-46s %-14s %6s %+7.0f\n", "-", "(no routers answer)", "", "", st.Ms)
		}
		for _, h := range s.Hops {
			if st.FirstTTL > 0 && h.TTL >= st.FirstTTL && h.TTL <= st.LastTTL {
				b.WriteString(row(fmt.Sprint(h.TTL), h))
			}
		}
		if st.Kind == "site" {
			b.WriteString(row("end", s.Dest))
		}
	}
}

func writeChanges(b *strings.Builder, changes []RouteChange) {
	if len(changes) == 0 {
		return
	}
	b.WriteString("\nRoute changes:\n")
	for _, c := range changes {
		fmt.Fprintf(b, "  %s  hop %d: %s -> %s\n", time.UnixMilli(c.T).Format("15:04:05"), c.TTL, c.From, c.To)
	}
}

func row(label string, h Hop) string {
	where := "-"
	if h.Place != nil {
		where = h.Place.City
	}
	if !h.Answered {
		return fmt.Sprintf("%3s  %-46s %-14s\n", label, "(doesn't answer pings)", where)
	}
	router := strings.TrimSpace(h.Addr + " " + h.Name)
	if len(router) > 46 {
		router = router[:45] + "…"
	}
	note := ""
	if h.Ignores {
		note = "  ignores pings"
	}
	return fmt.Sprintf("%3s  %-46s %-14s %5.1f%% %7.1f %7.1f%s\n", label, router, where, h.Loss, h.Avg, h.Jitter, note)
}

func nonEmpty(parts ...string) []string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func span(from, to int64) string {
	a, z := time.UnixMilli(from), time.UnixMilli(to)
	return a.Format("2006-01-02 15:04:05") + "–" + z.Format("15:04:05 MST")
}
