package probe

import (
	"context"
	"math/rand/v2"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"
)

// pathMemory is how long a hop that reached the destination keeps the route length fixed.
const pathMemory = 30 * time.Second

type pending struct {
	dst  netip.Addr
	ttl  int
	sent time.Time
}

// Engine schedules probes for several targets on one Sender and turns replies into Results.
type Engine struct {
	s       Sender
	out     func(Result)
	mu      sync.Mutex
	seq     uint16
	pending map[uint16]pending
	reached map[netip.Addr]*[MaxTTL + 1]time.Time
	targets atomic.Int32
}

// NewEngine returns an Engine that sends on s and passes every Result to out.
func NewEngine(s Sender, out func(Result)) *Engine {
	return &Engine{
		s:       s,
		out:     out,
		seq:     uint16(rand.N(65536)),
		pending: map[uint16]pending{},
		reached: map[netip.Addr]*[MaxTTL + 1]time.Time{},
	}
}

// Run matches replies and expires lost probes until ctx ends.
func (e *Engine) Run(ctx context.Context) {
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-e.s.Replies():
			e.match(r)
		case now := <-tick.C:
			e.expire(now)
		}
	}
}

// Trace probes the destination at the sender's rate and each hop toward dst once every
// n seconds, where n is the number of targets being traced. Shared nearby routers then
// get one probe a second in total, below the reply limit many of them enforce per sender.
func (e *Engine) Trace(ctx context.Context, dst netip.Addr, port int) {
	e.targets.Add(1)
	defer e.targets.Add(-1)
	for ctx.Err() == nil {
		n := int(e.targets.Load())
		slots := schedule(e.PathLen(dst), e.s.Mode().DestPerSecond*n)
		gap := time.Duration(n) * time.Second / time.Duration(len(slots))
		for _, ttl := range slots {
			e.send(Packet{Dst: dst, TTL: ttl, Port: port})
			if !sleep(ctx, gap) {
				return
			}
		}
	}
}

// PathLen returns the lowest hop limit that recently reached dst, or MaxTTL.
func (e *Engine) PathLen(dst netip.Addr) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	r := e.reached[dst]
	if r == nil {
		return MaxTTL
	}
	for ttl := 1; ttl <= MaxTTL; ttl++ {
		if time.Since(r[ttl]) < pathMemory {
			return ttl
		}
	}
	return MaxTTL
}

// schedule interleaves hop TTLs 1..pathLen with destRate destination probes.
func schedule(pathLen, destRate int) []int {
	slots := make([]int, 0, pathLen+destRate)
	for ttl := 1; ttl <= pathLen; ttl++ {
		slots = append(slots, ttl)
	}
	for d := 0; d < destRate; d++ {
		at := d * (len(slots) + 1) / destRate
		slots = append(slots[:at], append([]int{DestTTL}, slots[at:]...)...)
	}
	return slots
}

func (e *Engine) send(p Packet) {
	e.mu.Lock()
	p.Seq = e.nextSeq()
	e.pending[p.Seq] = pending{dst: p.Dst, ttl: p.TTL, sent: time.Now()}
	e.mu.Unlock()
	_ = e.s.Send(p) // a failed send expires as a lost probe
}

func (e *Engine) nextSeq() uint16 {
	for {
		e.seq++
		if _, busy := e.pending[e.seq]; !busy {
			return e.seq
		}
	}
}

func (e *Engine) match(r Reply) {
	e.mu.Lock()
	p, ok := e.pending[r.Seq]
	if !ok || p.dst != r.Dst || !answers(p, r) {
		e.mu.Unlock()
		return
	}
	delete(e.pending, r.Seq)
	if r.From == p.dst && p.ttl <= MaxTTL {
		e.markReached(p.dst, p.ttl, r.At)
	}
	e.mu.Unlock()
	e.out(Result{Dst: p.dst, TTL: p.ttl, Sent: p.sent, From: r.From, RTT: r.At.Sub(p.sent)})
}

// answers reports whether r answers probe p. Anything from the destination does, a refusal
// included. A router answers only a hop probe, and only by reporting the hop limit expired;
// a router's unreachable error means the probe got no further, so the probe expires as lost.
func answers(p pending, r Reply) bool {
	if r.From == p.dst {
		return true
	}
	return p.ttl != DestTTL && !r.Unreachable
}

func (e *Engine) markReached(dst netip.Addr, ttl int, at time.Time) {
	r := e.reached[dst]
	if r == nil {
		r = &[MaxTTL + 1]time.Time{}
		e.reached[dst] = r
	}
	r[ttl] = at
}

func (e *Engine) expire(now time.Time) {
	var lost []Result
	e.mu.Lock()
	for seq, p := range e.pending {
		if now.Sub(p.sent) > Timeout {
			delete(e.pending, seq)
			lost = append(lost, Result{Dst: p.dst, TTL: p.ttl, Sent: p.sent, Lost: true})
		}
	}
	e.mu.Unlock()
	for _, r := range lost {
		e.out(r)
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
