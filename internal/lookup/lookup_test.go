package lookup

import (
	"net/netip"
	"testing"
)

func TestOriginQueryReversesOctets(t *testing.T) {
	got := originQuery(netip.MustParseAddr("198.51.100.7"))
	if got != "7.100.51.198.origin.asn.cymru.com" {
		t.Fatal(got)
	}
}

func TestParseOrigin(t *testing.T) {
	cases := map[string]int{
		"64496 | 192.0.2.0/24 | US | arin | 2012-05-01":   64496,
		"64497 64498 | 198.51.100.0/24 | US | ripencc | ": 64497,
		"": 0,
	}
	for in, want := range cases {
		if got := parseOrigin(in); got != want {
			t.Errorf("parseOrigin(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseASName(t *testing.T) {
	cases := map[string]string{
		"64496 | US | arin | 2001-06-29 | EXAMPLE-ISP, US": "EXAMPLE-ISP",
		"174 | US | arin | 2001-04-20 | COGENT-174, US":    "COGENT-174",
		"64500 | ZZ | x | y | EXAMPLE":                     "EXAMPLE",
	}
	for in, want := range cases {
		if got := parseASName(in); got != want {
			t.Errorf("parseASName(%q) = %q, want %q", in, got, want)
		}
	}
}
