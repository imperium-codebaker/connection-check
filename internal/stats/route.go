package stats

import (
	"sync"
	"time"
)

// Route holds the samples for every hop toward one destination plus the destination itself.
type Route struct {
	mu   sync.Mutex
	hops map[int]*Series
	dest Series
}

// NewRoute returns an empty Route.
func NewRoute() *Route {
	return &Route{hops: map[int]*Series{}}
}

// AddHop records a sample for the hop at ttl.
func (r *Route) AddHop(ttl int, x Sample) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.hops[ttl]
	if s == nil {
		s = &Series{}
		r.hops[ttl] = s
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
