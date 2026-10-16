// Package monitor runs the probes for each site and assembles what the page shows.
package monitor

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"runtime"
	"sync"
	"time"

	"github.com/johncbrubaker/connection-check/internal/lookup"
	"github.com/johncbrubaker/connection-check/internal/probe"
	"github.com/johncbrubaker/connection-check/internal/report"
	"github.com/johncbrubaker/connection-check/internal/sites"
	"github.com/johncbrubaker/connection-check/internal/stats"
)

// Window kinds the page can choose.
const (
	Default = "default" // the last 5 minutes
	Bad     = "bad"     // 30 s either side of the moment the user pressed "It's bad right now"
	Custom  = "custom"  // a range dragged on the timeline
)

const (
	defaultSpan  = 5 * time.Minute
	badHalf      = 30 * time.Second
	timelineSpan = 15 * time.Minute
	bucketWidth  = 10 * time.Second
)

// window is the time range a site's verdict describes.
type window struct {
	kind     string
	from, to time.Time
}

// span returns the window's range as of now.
func (w window) span(now time.Time) (time.Time, time.Time) {
	switch w.kind {
	case Bad:
		return w.from, minTime(w.to, now)
	case Custom:
		return w.from, w.to
	}
	return now.Add(-defaultSpan), now
}

type site struct {
	cfg    sites.Site
	addr   netip.Addr
	route  *stats.Route
	window window
}

// Monitor probes every site on one engine and keeps their samples.
type Monitor struct {
	ctx   context.Context
	eng   *probe.Engine
	mode  probe.Mode
	look  *lookup.Resolver
	mu    sync.Mutex
	sites []*site
}

// New starts an engine on sender that runs until ctx ends.
func New(ctx context.Context, sender probe.Sender) *Monitor {
	m := &Monitor{ctx: ctx, mode: sender.Mode(), look: lookup.New()}
	m.eng = probe.NewEngine(sender, m.record)
	go m.eng.Run(ctx)
	return m
}

// Add resolves a site's host and starts tracing it.
func (m *Monitor) Add(cfg sites.Site) error {
	addr, err := resolve(m.ctx, cfg.Host)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.sites = append(m.sites, &site{cfg: cfg, addr: addr, route: stats.NewRoute(), window: window{kind: Default}})
	m.mu.Unlock()
	go m.eng.Trace(m.ctx, addr, cfg.Port)
	return nil
}

// Sites returns the configured sites.
func (m *Monitor) Sites() []sites.Site {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]sites.Site, len(m.sites))
	for i, s := range m.sites {
		out[i] = s.cfg
	}
	return out
}

// SetWindow chooses the window for site i. For Bad the moment is now; for Custom, from and to are used.
func (m *Monitor) SetWindow(i int, kind string, from, to time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if i < 0 || i >= len(m.sites) {
		return fmt.Errorf("no site %d", i)
	}
	w, err := newWindow(kind, from, to, time.Now())
	if err == nil {
		m.sites[i].window = w
	}
	return err
}

func newWindow(kind string, from, to, now time.Time) (window, error) {
	switch kind {
	case Default:
		return window{kind: Default}, nil
	case Bad:
		return window{kind: Bad, from: now.Add(-badHalf), to: now.Add(badHalf)}, nil
	case Custom:
		if !from.Before(to) {
			return window{}, fmt.Errorf("window ends before it starts")
		}
		return window{kind: Custom, from: from, to: to}, nil
	}
	return window{}, fmt.Errorf("unknown window %q", kind)
}

// Snapshots returns every site's current snapshot.
func (m *Monitor) Snapshots() []report.Snapshot {
	m.mu.Lock()
	list := append([]*site(nil), m.sites...)
	m.mu.Unlock()
	now := time.Now()
	out := make([]report.Snapshot, len(list))
	for i, s := range list {
		out[i] = m.snapshot(s, now)
	}
	return out
}

// record files one probe result under every site traced at its destination
// and starts looking up the responder's name and network.
func (m *Monitor) record(r probe.Result) {
	x := stats.Sample{At: r.Sent, RTT: r.RTT, Lost: r.Lost, From: r.From}
	if !r.Lost {
		m.look.Get(r.From)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sites {
		switch {
		case s.addr != r.Dst:
		case r.TTL == probe.DestTTL:
			s.route.AddDest(x)
		default:
			s.route.AddHop(r.TTL, x)
		}
	}
}

func (m *Monitor) modeInfo() report.Mode {
	return report.Mode{Name: m.mode.Name, Partial: m.mode.Partial, Hint: m.mode.Hint}
}

func resolve(ctx context.Context, host string) (netip.Addr, error) {
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", host)
	if err != nil {
		return netip.Addr{}, err
	}
	if len(addrs) == 0 {
		return netip.Addr{}, fmt.Errorf("%s has no IPv4 address", host)
	}
	return addrs[0].Unmap(), nil
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

var goos = runtime.GOOS
