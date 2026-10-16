package geo

import (
	"strings"
	"testing"
)

// placeOne places a single named hop at the given round trip.
func placeOne(name string, rtt float64) *Place {
	return PlaceRoute([]Hop{{Name: name, MinRTT: rtt}})[0]
}

// cityOf returns a place's city, or "" for nil.
func cityOf(p *Place) string {
	if p == nil {
		return ""
	}
	return p.City
}

// checkRoute places hops and compares each hop's city with want: a city prefix, or "" for unplaced.
func checkRoute(t *testing.T, hops []Hop, want []string) []*Place {
	t.Helper()
	got := PlaceRoute(hops)
	for i, w := range want {
		if c := cityOf(got[i]); !strings.HasPrefix(c, w) || w == "" && c != "" {
			t.Errorf("%v hop %d = %q, want %q", hops, i, c, w)
		}
	}
	return got
}

func TestSingleNames(t *testing.T) {
	tests := []struct {
		name string
		want string // prefix of the city, "" for unplaced
	}{
		{"ae2---0.scr05.lsan.ca.example-isp.net", "Los Angeles, CA"},
		{"ae0---0.cbr21.lsan.ca.example-isp.net", "Los Angeles, CA"},
		{"ae0.cr10-lax2.ip4.gtt.net", "Los Angeles, CA"},
		{"be2199.ccr42.lax01.atlas.cogentco.com", "Los Angeles, CA"},
		{"be2932.ccr32.phx01.atlas.cogentco.com", "Phoenix, AZ"},
		{"be5473.ccr22.elp02.atlas.cogentco.com", "El Paso, TX"},
		{"be3846.ccr32.dfw01.atlas.cogentco.com", "Dallas"},
		{"port-channel8122.ccr92.jan02.atlas.cogentco.com", "Jackson, MS"},
		{"be3704.ccr42.atl01.atlas.cogentco.com", "Atlanta, GA"},
		{"ae36---100.ber01.qqzz.ca.example-isp.net", ""},
		{"203-0-113-1.bng01.qqzz.ca.example-isp.net", ""},
		{"3257.customer.alter.net", ""},
		{"ip4.gtt.net", ""},
		{"lsanca07", "Los Angeles, CA"},
		{"dllstx", "Dallas, TX"},
		{"nycmny", "New York, NY"},
		{"xe-0-0-1.nycmnyqo.example.net", "New York, NY"},
		{"router.example.net", ""},
		{"", ""},
		{"ber", ""},
		{"ber01", ""},
	}
	for _, tt := range tests {
		checkRoute(t, []Hop{{tt.name, 10}}, []string{tt.want})
	}
}

func TestStrongMatchesAreExact(t *testing.T) {
	for _, name := range []string{"lsan.ca.example.net", "be1.lax01.example.net", "lsanca07"} {
		if p := placeOne(name, 10); p == nil || p.Approx {
			t.Errorf("%q gave %+v, want an exact place", name, p)
		}
	}
}

func TestBareTokens(t *testing.T) {
	tests := []struct {
		hops []Hop
		want []string
	}{
		{[]Hop{{"phx.example.net", 10}}, []string{"Phoenix, AZ"}},
		{[]Hop{{"phx.example.net", 0}}, []string{""}},
		{[]Hop{{"be1.lax01.example.net", 2}, {"lax.example.net", 3}}, []string{"Los Angeles, CA", "Los Angeles, CA"}},
		{[]Hop{{"be1.lax01.example.net", 2}, {"sfo.example.net", 30}}, []string{"Los Angeles, CA", ""}},
	}
	for _, tt := range tests {
		got := checkRoute(t, tt.hops, tt.want)
		if last := got[len(got)-1]; last != nil && !last.Approx {
			t.Errorf("%v: bare token placed exactly", tt.hops)
		}
	}
}

func TestLatencyRejectsImpossibleHops(t *testing.T) {
	tests := []struct {
		hops []Hop
		want []string
	}{
		{[]Hop{{"be1.lax01.example.net", 2}, {"xe-1.lhr01.example.net", 4}}, []string{"Los Angeles, CA", ""}},
		{[]Hop{{"be1.lax01.example.net", 2}, {"xe-1.lhr01.example.net", 0}}, []string{"Los Angeles, CA", "London, GB"}},
		{[]Hop{{"lsan.ca.example.net", 1}, {"be1.sfo01.example.net", 4}}, []string{"Los Angeles, CA", ""}},
		{[]Hop{{"be1.lax01.example.net", 2}, {"be1.jfk01.example.net", 70}}, []string{"Los Angeles, CA", "New York"}},
	}
	for _, tt := range tests {
		checkRoute(t, tt.hops, tt.want)
	}
}

func TestRealisticRoute(t *testing.T) {
	hops := []Hop{
		{"", 1},
		{"be2199.ccr42.lax01.atlas.cogentco.com", 2},
		{"be2932.ccr32.phx01.atlas.cogentco.com", 13},
		{"be5473.ccr22.elp02.atlas.cogentco.com", 23},
		{"be3846.ccr32.dfw01.atlas.cogentco.com", 32},
		{"port-channel8122.ccr92.jan02.atlas.cogentco.com", 39},
		{"be3704.ccr42.atl01.atlas.cogentco.com", 47},
	}
	want := []string{"", "Los Angeles, CA", "Phoenix, AZ", "El Paso, TX", "Dallas", "Jackson, MS", "Atlanta, GA"}
	for _, p := range checkRoute(t, hops, want)[1:] {
		if p != nil && p.Approx {
			t.Errorf("%s placed approximately", p.City)
		}
	}
}

func TestSlowAnsweringRouterDoesNotBlockLaterHops(t *testing.T) {
	hops := []Hop{
		{"ae2---0.scr05.lsan.ca.example-isp.net", 14},
		{"ae0---0.cbr06.lsan.ca.example-isp.net", 1.6},
		{"be2199.ccr42.lax01.atlas.cogentco.com", 1.8},
		{"be2932.ccr32.phx01.atlas.cogentco.com", 12.3},
		{"be5473.ccr22.elp02.atlas.cogentco.com", 22},
	}
	want := []string{"Los Angeles, CA", "Los Angeles, CA", "Los Angeles, CA", "Phoenix, AZ", "El Paso, TX"}
	checkRoute(t, hops, want)
}

func TestTables(t *testing.T) {
	if len(iataSites) < 1000 || len(clliSites) < 70 {
		t.Fatalf("tables too small: %d IATA, %d CLLI", len(iataSites), len(clliSites))
	}
	for code := range clliSites {
		if len(code) != 6 || !allLetters(code) || !usStates[code[4:]] {
			t.Errorf("CLLI code %q is not four letters plus a US state", code)
		}
	}
}

func TestDistanceKm(t *testing.T) {
	d := distanceKm(iataSites["lax"], iataSites["jfk"])
	if d < 3900 || d > 4050 {
		t.Errorf("LAX-JFK = %.0f km, want about 3980", d)
	}
}
