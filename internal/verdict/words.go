package verdict

import (
	"fmt"

	"github.com/johncbrubaker/connection-check/internal/owner"
)

// words returns the headline and suggested action for v; located is false when the hops
// got too few pings to tell where the trouble starts.
func words(in Input, v Verdict, located bool) (string, string) {
	if v.Grade == Good {
		return fmt.Sprintf("Your connection to %s looks fine (%.0f ms, %.1f%% loss).", in.Site, in.Dest.Avg, in.Dest.Loss),
			"If a session still stutters, press “It's bad right now” the moment it happens."
	}
	if !located && in.Dest.Loss >= 100 {
		return fmt.Sprintf("Nothing is getting through to %s.", in.Site),
			"If other websites don't load either, restart your router or call your internet provider. " +
				"If they do, send this report to IT."
	}
	if !located {
		return headline(in, v), "Too few pings in this time range to tell where it starts. " +
			"Leave Connection Check running or pick a longer range, then send this report to IT."
	}
	return headline(in, v), action(in, v)
}

func headline(in Input, v Verdict) string {
	d := in.Dest
	switch {
	case v.Measure == Loss && v.Grade == Poor:
		return fmt.Sprintf("Your connection to %s is losing packets (%.1f%%).", in.Site, d.Loss)
	case v.Measure == Loss:
		return fmt.Sprintf("Your connection to %s is dropping a few packets (%.2f%%).", in.Site, d.Loss)
	case v.Measure == Jitter:
		return fmt.Sprintf("Your connection to %s is unsteady (timing varies by %.0f ms).", in.Site, d.Jitter)
	case v.Grade == Poor:
		return fmt.Sprintf("Your connection to %s is slow (%.0f ms).", in.Site, d.Avg)
	}
	return fmt.Sprintf("Your connection to %s is usable but slow (%.0f ms).", in.Site, d.Avg)
}

func action(in Input, v Verdict) string {
	if v.LongRoad {
		return fmt.Sprintf("The delay comes from the long route between your provider and %s, not a fault. "+
			"Nothing to fix at home; send this report to IT, who may be able to shorten the route.", in.Site)
	}
	switch v.Owner {
	case owner.Home:
		return "It starts at your own router or Wi-Fi. Try a network cable, move closer to the router, or restart it."
	case owner.Provider:
		return "It starts inside your internet provider's network" + named(in.Provider) +
			". Call them and share this report."
	case owner.Handoff:
		return fmt.Sprintf("It starts between your provider and %s, on another carrier's network. "+
			"Send this report to IT.", in.Site)
	case owner.Site:
		return fmt.Sprintf("It starts at %s's end. Nothing to fix at home; send this report to IT.", in.Site)
	}
	return "It builds up along the route rather than at one place. Send this report to IT."
}

func named(provider string) string {
	if provider == "" {
		return ""
	}
	return " (" + provider + ")"
}
