package stats

import (
	"net/netip"
	"slices"
	"sync"
	"time"
)

// KeepChanges is how many route changes a Route remembers.
const KeepChanges = 20

// Route holds the samples for every hop toward one destination plus the destination itself.
type Route struct {
	mu      sync.Mutex
	hops    map[int]*Series
	dest    Series
	changes []Change
}

// Change records the hop at TTL answering from a different router.
type Change struct {
	At       time.Time
	TTL      int
	From, To netip.Addr
}

// NewRoute returns an empty Route.
func NewRoute() *Route {
	return &Route{hops: map[int]*Series{}}
}

// AddHop records a sample for the hop at ttl. A reply from a different router than the one
// answering there so far starts the hop afresh, dropping the old router's samples, so one
// row never mixes two routers; the swap is kept as a Change.
func (r *Route) AddHop(ttl int, x Sample) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.hops[ttl]
	if s != nil && !x.Lost && s.router.IsValid() && x.From != s.router {
		r.changes = append(r.changes, Change{At: x.At, TTL: ttl, From: s.router, To: x.From})
		if len(r.changes) > KeepChanges {
			r.changes = slices.Delete(r.changes, 0, len(r.changes)-KeepChanges)
		}
		s = nil
	}
	if s == nil {
		s = &Series{}
		r.hops[ttl] = s
	}
	if !x.Lost {
		s.router = x.From
	}
	s.Add(x)
}

// AddDest records a destination sample.
func (r *Route) AddDest(x Sample) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dest.Add(x)
}

// Hops summarises hops 1..n over [from, to); index 0 is hop 1.
func (r *Route) Hops(n int, from, to time.Time) []Summary {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Summary, n)
	for ttl := 1; ttl <= n; ttl++ {
		if s := r.hops[ttl]; s != nil {
			out[ttl-1] = s.Summary(from, to)
		}
	}
	return out
}

// Dest summarises the destination over [from, to).
func (r *Route) Dest(from, to time.Time) Summary {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.dest.Summary(from, to)
}

// Timeline returns destination buckets of width covering [from, to).
func (r *Route) Timeline(from, to time.Time, width time.Duration) []Bucket {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.dest.Buckets(from, to, width)
}

// Changes returns the remembered route changes, oldest first.
func (r *Route) Changes() []Change {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.changes)
}
