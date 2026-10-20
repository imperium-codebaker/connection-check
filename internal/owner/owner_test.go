package owner

import (
	"net/netip"
	"slices"
	"testing"
)

func hop(addr string, asn int) Hop {
	if addr == "" {
		return Hop{}
	}
	return Hop{Addr: netip.MustParseAddr(addr), ASN: asn}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		hops []Hop
		want []Owner
	}{
		{
			name: "typical route with silent hops",
			hops: []Hop{hop("192.168.1.1", 0), hop("198.51.100.1", 64500), hop("", 0),
				hop("198.51.100.9", 64500), hop("203.0.113.5", 64501), hop("", 0), hop("192.0.2.40", 64502)},
			want: []Owner{Home, Provider, Provider, Provider, Handoff, Unknown, Site},
		},
		{
			name: "double NAT at home and carrier NAT at the provider",
			hops: []Hop{hop("192.168.1.1", 0), hop("10.0.0.1", 0), hop("100.64.0.1", 0), hop("198.51.100.1", 64500)},
			want: []Owner{Home, Home, Provider, Provider},
		},
		{
			name: "private address inside a carrier takes its neighbours' label",
			hops: []Hop{hop("192.168.1.1", 0), hop("198.51.100.1", 64500), hop("203.0.113.5", 64501),
				hop("10.9.9.9", 0), hop("203.0.113.9", 64501)},
			want: []Owner{Home, Provider, Handoff, Handoff, Handoff},
		},
		{
			name: "silent first hop",
			hops: []Hop{hop("", 0), hop("198.51.100.1", 64500)},
			want: []Owner{Unknown, Provider},
		},
	}
	for _, c := range cases {
		if got := Classify(c.hops, 64502); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestShortName(t *testing.T) {
	cases := map[string]string{
		"EXAMPLE-BACKBONE - Example Communications Inc.":   "Example",
		"TRANSIT-AS - Sample Business":                     "Sample",
		"LEVEL9-AS - Level 9 Parent, LLC":                  "Level 9",
		"SAMPLE-BELL - Sample Bell Telephone Company LLC":  "Sample Bell",
		"ORBIT-AS - Orbit Launch Technologies Corporation": "Orbit Launch",
		"STATE-NET - State of Example":                     "State of Example",
		"EXAMPLE-ISP":                                      "EXAMPLE-ISP",
		"Example Networks, Inc.":                           "Example",
		"ODD-NAME - ":                                      "ODD-NAME",
		"":                                                 "",
	}
	for in, want := range cases {
		if got := ShortName(in); got != want {
			t.Errorf("ShortName(%q) = %q, want %q", in, got, want)
		}
	}
}
