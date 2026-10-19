package geo

import (
	_ "embed"
	"math"
	"strconv"
	"strings"
)

//go:embed airports.tsv
var airportsTSV string

//go:embed clli.tsv
var clliTSV string

// site is one entry of an embedded city-code table.
type site struct {
	City string
	Lat  float64
	Lon  float64
}

var (
	iataSites = parseTable(airportsTSV)
	clliSites = parseTable(clliTSV)
)

// usStates holds the two-letter postal abbreviations of the US states and DC.
var usStates = setOf("al ak az ar ca co ct de dc fl ga hi id il in ia ks ky la me md ma mi mn ms mo mt " +
	"ne nv nh nj nm ny nc nd oh ok or pa ri sc sd tn tx ut vt va wa wv wi wy")

// setOf returns the whitespace-separated words of s as a set.
func setOf(s string) map[string]bool {
	set := map[string]bool{}
	for _, w := range strings.Fields(s) {
		set[w] = true
	}
	return set
}

// parseTable reads tab-separated "code city lat lon" rows, skipping "#" comment
// lines and malformed rows, and returns them keyed by code.
func parseTable(data string) map[string]site {
	table := map[string]site{}
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSuffix(line, "\r") // a Windows checkout may convert the tables to CRLF
		f := strings.Split(line, "\t")
		if len(f) != 4 || strings.HasPrefix(line, "#") {
			continue
		}
		lat, errLat := strconv.ParseFloat(f[2], 64)
		lon, errLon := strconv.ParseFloat(f[3], 64)
		if errLat == nil && errLon == nil {
			table[f[0]] = site{City: f[1], Lat: lat, Lon: lon}
		}
	}
	return table
}

// distanceKm returns the great-circle distance between two sites in kilometres.
func distanceKm(a, b site) float64 {
	const earthRadiusKm = 6371.0
	rad := math.Pi / 180
	dLat := (b.Lat - a.Lat) * rad
	dLon := (b.Lon - a.Lon) * rad
	h := math.Pow(math.Sin(dLat/2), 2) +
		math.Cos(a.Lat*rad)*math.Cos(b.Lat*rad)*math.Pow(math.Sin(dLon/2), 2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(h)))
}
