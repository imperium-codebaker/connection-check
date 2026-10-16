package report

import (
	"fmt"
	"strings"
	"time"
)

var windowNames = map[string]string{
	"default": "last 5 minutes",
	"bad":     "marked as a bad moment",
	"custom":  "selected range",
}

// Text returns a plain-text summary of s for email or chat.
func Text(s Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Connection Check report: %s (%s, %s)\n", s.Site, s.Host, s.Addr)
	fmt.Fprintf(&b, "Window: %s (%s), %s, %s probes\n", span(s.From, s.To), windowNames[s.Window], s.OS, s.Mode.Name)
	fmt.Fprintf(&b, "Verdict: %s. %s\n%s\n\n", strings.ToUpper(s.Verdict.Grade), s.Verdict.Headline, s.Verdict.Action)
	fmt.Fprintf(&b, "%3s  %-9s %-46s %6s %7s %7s\n", "#", "Owner", "Router", "Loss", "Avg ms", "Jitter")
	for _, h := range s.Hops {
		b.WriteString(row(fmt.Sprint(h.TTL), h))
	}
	b.WriteString(row("end", s.Dest))
	return b.String()
}

func row(label string, h Hop) string {
	if !h.Answered {
		return fmt.Sprintf("%3s  %-9s %-46s\n", label, h.Owner, "(doesn't answer pings)")
	}
	router := strings.TrimSpace(h.Addr + " " + h.Name)
	if len(router) > 46 {
		router = router[:45] + "…"
	}
	return fmt.Sprintf("%3s  %-9s %-46s %5.1f%% %7.1f %7.1f\n", label, h.Owner, router, h.Loss, h.Avg, h.Jitter)
}

func span(from, to int64) string {
	a, z := time.UnixMilli(from), time.UnixMilli(to)
	return a.Format("2006-01-02 15:04:05") + "–" + z.Format("15:04:05 MST")
}
