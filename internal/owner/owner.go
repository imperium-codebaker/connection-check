// Package owner labels each hop with who runs it: home, provider, handoff carriers or the remote site.
package owner

import "net/netip"

// Owner names who runs a hop.
type Owner string

// Owners, in route order.
const (
	Home     Owner = "home"
	Provider Owner = "provider"
	Handoff  Owner = "handoff"
	Site     Owner = "site"
	Unknown  Owner = "unknown"
)

// Hop is what Classify needs to know about one hop.
type Hop struct {
	Addr netip.Addr // zero when the hop never answered
	ASN  int        // 0 when unknown or private
}

var carrierNAT = netip.MustParsePrefix("100.64.0.0/10")

// Classify labels each hop, given the destination's network number.
func Classify(hops []Hop, destASN int) []Owner {
	out := make([]Owner, len(hops))
	home := homeEnd(hops)
	provider := providerASN(hops[home:], destASN)
	for i, h := range hops {
		out[i] = direct(h, i < home, provider, destASN)
	}
	fillGaps(out)
	return out
}

// homeEnd returns the index of the first hop past the leading private addresses.
func homeEnd(hops []Hop) int {
	end := 0
	for i, h := range hops {
		if !h.Addr.IsValid() {
			continue
		}
		if !isPrivate(h.Addr) {
			return end
		}
		end = i + 1
	}
	return end
}

// providerASN returns the network number of the first public hop, unless it is the destination's.
func providerASN(hops []Hop, destASN int) int {
	for _, h := range hops {
		if h.ASN != 0 {
			if h.ASN == destASN {
				return 0
			}
			return h.ASN
		}
	}
	return 0
}

// direct labels a hop from its own address alone; Unknown means "decide from the neighbours".
func direct(h Hop, inHome bool, provider, destASN int) Owner {
	switch {
	case inHome && h.Addr.IsValid():
		return Home
	case !h.Addr.IsValid():
		return Unknown
	case carrierNAT.Contains(h.Addr):
		return Provider
	case h.ASN == 0:
		return Unknown
	case h.ASN == destASN:
		return Site
	case h.ASN == provider:
		return Provider
	}
	return Handoff
}

// fillGaps gives an Unknown hop its neighbours' label when the nearest known hops on both sides agree.
func fillGaps(out []Owner) {
	for i := range out {
		if out[i] != Unknown {
			continue
		}
		before, after := nearest(out, i, -1), nearest(out, i, 1)
		if before == after {
			out[i] = before
		}
	}
}

func nearest(out []Owner, i, step int) Owner {
	for j := i + step; j >= 0 && j < len(out); j += step {
		if out[j] != Unknown {
			return out[j]
		}
	}
	return Unknown
}

func isPrivate(a netip.Addr) bool {
	return a.IsPrivate() || a.IsLinkLocalUnicast() || a.IsLoopback()
}
