package monitor

import (
	"context"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/imperium-codebaker/connection-check/internal/lookup"
	"github.com/imperium-codebaker/connection-check/internal/probe"
	"github.com/imperium-codebaker/connection-check/internal/stats"
)

// Run under -race: a snapshot taken while the window changes must describe one window.
func TestSnapshotWhileWindowChanges(t *testing.T) {
	s := &site{addr: netip.MustParseAddr("127.0.0.1"), route: stats.NewRoute(), window: window{kind: Default, length: defaultSpan}}
	m := &Monitor{ctx: context.Background(), eng: probe.NewEngine(nil, nil), look: lookup.New(), sites: []*site{s}}
	from, to := time.Unix(1_700_000_000, 0), time.Unix(1_700_000_600, 0)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 200 {
			if i%2 == 0 {
				_ = m.SetWindow(0, Custom, from, to)
			} else {
				_ = m.SetWindow(0, Default, time.Time{}, time.Time{})
			}
		}
	}()
	for range 200 {
		snap := m.Snapshots()[0]
		if custom := snap.From == from.UnixMilli() && snap.To == to.UnixMilli(); custom != (snap.Window == Custom) {
			t.Fatalf("window %q labels range %d..%d", snap.Window, snap.From, snap.To)
		}
	}
	wg.Wait()
}

func TestWindowSpans(t *testing.T) {
	press := time.Unix(1_700_000_000, 0)
	bad, _ := newWindow(Bad, time.Time{}, time.Time{}, press)
	if from, to := bad.span(press.Add(10 * time.Second)); !from.Equal(press.Add(-badHalf)) || !to.Equal(press.Add(10*time.Second)) {
		t.Errorf("bad window while running = %v..%v", from, to)
	}
	if _, to := bad.span(press.Add(time.Hour)); !to.Equal(press.Add(badHalf)) {
		t.Errorf("bad window should end 30 s after the press, ended %v", to)
	}
	def, _ := newWindow(Default, time.Time{}, time.Time{}, press)
	if from, _ := def.span(press); !from.Equal(press.Add(-defaultSpan)) {
		t.Errorf("default window starts %v", from)
	}
	minute, _ := newWindow(Minute, time.Time{}, time.Time{}, press)
	if from, to := minute.span(press); !from.Equal(press.Add(-time.Minute)) || !to.Equal(press) {
		t.Errorf("minute window = %v..%v", from, to)
	}
	all, _ := newWindow(All, press.Add(-15*time.Minute), press, press)
	if from, _ := all.span(press.Add(time.Hour)); !from.Equal(press.Add(45 * time.Minute)) {
		t.Errorf("15-minute whole-chart window should follow now, starts %v", from)
	}
	if long, _ := newWindow(All, press.Add(-6*time.Hour), press, press); long.length != timelineSpan {
		t.Errorf("whole-chart window reaches back %v, longer than the chart", long.length)
	}
	if _, err := newWindow(Custom, press, press.Add(-time.Second), press); err == nil {
		t.Error("backwards custom window accepted")
	}
	if _, err := newWindow("nonsense", press, press, press); err == nil {
		t.Error("unknown window accepted")
	}
}
