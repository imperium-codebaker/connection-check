package geo

import "strings"

// strength ranks how strongly a router-name token identifies a city.
type strength int

const (
	// bareIATA is a lone three-letter token that matches an airport code.
	bareIATA strength = iota + 1
	// codedIATA is a three-letter airport code with a site number attached, such as "lax01".
	codedIATA
	// clliCity is a CLLI city+state code, such as "lsanca07" or "lsan" followed by "ca".
	clliCity
)

// candidate is one possible location for a hop.
type candidate struct {
	hop      int
	site     site
	strength strength
}

// token is one piece of a router name with trailing digits removed.
type token struct {
	text      string
	hadDigits bool
}

// stopWords are router-function and hosting words that must never be read as places.
var stopWords = setOf("ae be xe et ge te hu fe gi po ve fa si lo eth gig ten tge hge pos atm ser tun " +
	"cr ccr scr cbr ber bng bb bbr icr mcr rcr ear bar car agr sar per csr asr mpr bdr brd " +
	"core edge agg aggr acc access dist ar br gw rtr pe p rr mx pr cs ds sw ip net com org " +
	"customer cust static dynamic dsl cable res atlas port channel vlan irb bvi lag bun " +
	"int ext srv mgmt oob isp bgp ix ixp peer transit link trunk cpe olt lns bras pool " +
	"host user client dial ppp fiber gpon mpls wan lan nat vpn fw lb hsd hfc cmts ubr " +
	"rev ptr arpa www mail ns dns dhcp tel inf")

// candidatesFor returns the places a router name could point at, in name order.
func candidatesFor(hop int, name string) []candidate {
	toks := tokenise(name)
	var found []candidate
	for i, t := range toks {
		if c, ok := matchToken(toks, i); ok && !stopWords[t.text] {
			c.hop = hop
			found = append(found, c)
		}
	}
	return found
}

// matchToken looks up token i of toks in the city-code tables.
func matchToken(toks []token, i int) (candidate, bool) {
	t := toks[i].text
	if s, ok := clliPrefix(t); ok {
		return candidate{site: s, strength: clliCity}, true
	}
	if len(t) == 4 && i+1 < len(toks) && usStates[toks[i+1].text] {
		s, ok := clliSites[t+toks[i+1].text]
		return candidate{site: s, strength: clliCity}, ok
	}
	if s, ok := iataSites[t]; ok && len(t) == 3 {
		if toks[i].hadDigits {
			return candidate{site: s, strength: codedIATA}, true
		}
		return candidate{site: s, strength: bareIATA}, true
	}
	return candidate{}, false
}

// clliPrefix matches a six-letter CLLI city+state token, or an eight-letter
// CLLI building token, against the CLLI table.
func clliPrefix(t string) (site, bool) {
	if (len(t) != 6 && len(t) != 8) || !allLetters(t) {
		return site{}, false
	}
	s, ok := clliSites[t[:6]]
	return s, ok
}

// tokenise lowercases a DNS name, drops its registered domain, splits the
// rest on ".", "-" and "_", strips trailing digits, and returns the non-empty tokens.
func tokenise(name string) []token {
	labels := hostLabels(name)
	var toks []token
	for _, piece := range strings.FieldsFunc(strings.Join(labels, "."), isSeparator) {
		text := strings.TrimRight(piece, "0123456789")
		if text != "" {
			toks = append(toks, token{text: text, hadDigits: text != piece})
		}
	}
	return toks
}

// hostLabels returns the labels of a DNS name without the registered domain:
// the last two labels of a longer name, or the TLD of a two-label name.
func hostLabels(name string) []string {
	labels := strings.Split(strings.Trim(strings.ToLower(name), "."), ".")
	switch {
	case len(labels) >= 3:
		return labels[:len(labels)-2]
	case len(labels) == 2:
		return labels[:1]
	}
	return labels
}

// isSeparator reports whether r splits a DNS label into tokens.
func isSeparator(r rune) bool {
	return r == '.' || r == '-' || r == '_'
}

// allLetters reports whether s holds only ASCII lowercase letters.
func allLetters(s string) bool {
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}
