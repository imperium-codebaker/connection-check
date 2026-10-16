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
	Window   string   `json:"window"` // "default", "bad" or "custom"
	Verdict  Verdict  `json:"verdict"`
	Hops     []Hop    `json:"hops"`
	Dest     Hop      `json:"dest"`
	Timeline []Bucket `json:"timeline"`
}

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
	Loss     float64 `json:"loss"`   // percent
	Avg      float64 `json:"avg"`    // ms
	Best     float64 `json:"best"`   // ms
	Worst    float64 `json:"worst"`  // ms
	Jitter   float64 `json:"jitter"` // ms, mean difference between consecutive replies
	Grade    string  `json:"grade"`  // "good", "fair", "poor" or "" when unanswered
	Place    *Place  `json:"place"`
}

// Place is a hop's estimated location.
type Place struct {
	City   string  `json:"city"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Approx bool    `json:"approx"`
}

// Bucket summarises destination probes over one timeline slice.
type Bucket struct {
	T    int64   `json:"t"` // slice start, Unix ms
	Sent int     `json:"sent"`
	Lost int     `json:"lost"`
	Avg  float64 `json:"avg"` // ms; 0 when every probe was lost
}
