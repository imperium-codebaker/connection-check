// Package report defines the snapshot shown by the page and carried in report links.
package report

// Snapshot is one site's state over a chosen time window.
// The live page and a decoded report link render the same shape.
type Snapshot struct {
	V        int      `json:"v"`
	Site     string   `json:"site"`
	Host     string   `json:"host"`
	Addr     string   `json:"addr"`
	From     int64    `json:"from"` // window start, Unix ms
	To       int64    `json:"to"`   // window end, Unix ms
	OS       string   `json:"os"`
	Mode     Mode     `json:"mode"`
	Window   string   `json:"window"` // "default", "minute", "all", "bad" or "custom"
	Verdict  Verdict  `json:"verdict"`
	Hops     []Hop    `json:"hops"`
	Dest     Hop      `json:"dest"`
	Timeline []Bucket `json:"timeline"`

	Stretches    []Stretch     `json:"stretches"`    // the route in stretches; absent in version 1
	RouteChanges []RouteChange `json:"routeChanges"` // most recent last; absent in version 1
}

// Version is the snapshot shape this program writes.
const Version = 2

// Mode describes how the probes were sent.
type Mode struct {
	Name    string `json:"name"`    // "icmp" or "udp+tcp"
	Partial bool   `json:"partial"` // true when far hops may not answer
	Hint    string `json:"hint,omitempty"`
}

// Verdict is the plain-English judgement for the window.
type Verdict struct {
	Grade    string `json:"grade"`   // "good", "fair", "poor" or "measuring"
	Measure  string `json:"measure"` // "loss", "latency", "jitter" or ""
	Owner    string `json:"owner"`   // "home", "provider", "handoff", "site" or ""
	BlameTTL int    `json:"blameTtl"`
	LongRoad bool   `json:"longRoad"`
	Headline string `json:"headline"`
	Action   string `json:"action"`
}

// Hop is one row of the hop table; Dest uses the same shape with TTL 0.
type Hop struct {
	TTL      int     `json:"ttl"`
	Addr     string  `json:"addr"` // "" when the hop never answered
	Name     string  `json:"name"`
	ASN      int     `json:"asn"`
	ASName   string  `json:"asName"`
	Owner    string  `json:"owner"` // "home", "provider", "handoff", "site" or "unknown"
	Answered bool    `json:"answered"`
	Sent     int     `json:"sent"`
	Lost     int     `json:"lost"`
	Loss     float64 `json:"loss"`    // percent
	Avg      float64 `json:"avg"`     // ms
	Best     float64 `json:"best"`    // ms
	Worst    float64 `json:"worst"`   // ms
	Jitter   float64 `json:"jitter"`  // ms, mean difference between consecutive replies
	Grade    string  `json:"grade"`   // "good", "fair", "poor" or "" when unanswered
	Ignores  bool    `json:"ignores"` // loses pings itself but passes no loss on
	Place    *Place  `json:"place"`
}

// Place is a hop's estimated location.
type Place struct {
	City   string  `json:"city"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Approx bool    `json:"approx"`
	Given  bool    `json:"given,omitempty"` // set in the site list, not found from the route
}

// Bucket summarises destination probes over one timeline slice.
type Bucket struct {
	T      int64   `json:"t"` // slice start, Unix ms
	Sent   int     `json:"sent"`
	Lost   int     `json:"lost"`
	Avg    float64 `json:"avg"`    // ms; 0 when every probe was lost
	Jitter float64 `json:"jitter"` // ms, mean difference between consecutive replies
}

// Stretch is a run of the route run by one kind of network, or the unseen link before the site.
type Stretch struct {
	Kind     string  `json:"kind"`     // "home", "provider", "handoff", "link" or "site"
	Name     string  `json:"name"`     // short network names, or "{city} to {city}" for a link
	City     string  `json:"city"`     // where the stretch ends, "" when unknown
	Approx   bool    `json:"approx"`   // the city is a guess
	FirstTTL int     `json:"firstTtl"` // first hop in the stretch; 0 when it holds only the destination or nothing
	LastTTL  int     `json:"lastTtl"`  // last hop before the destination in the stretch, 0 when none
	Routers  int     `json:"routers"`  // table rows in the stretch, the destination included
	Ms       float64 `json:"ms"`       // round trip at the stretch's end; for a link, the delay it adds
	Grade    string  `json:"grade"`    // worst grade in the stretch, "" when nothing is graded
	Blamed   bool    `json:"blamed"`   // the trouble starts in this stretch
	Near     bool    `json:"near"`     // the stretch ends close to the user
}

// RouteChange records a hop answering from a different router than before.
type RouteChange struct {
	T    int64  `json:"t"` // Unix ms
	TTL  int    `json:"ttl"`
	From string `json:"from"`
	To   string `json:"to"`
}
