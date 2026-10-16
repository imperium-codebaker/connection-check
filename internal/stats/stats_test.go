package stats

import (
	"math"
	"net/netip"
	"testing"
	"time"
)

var t0 = time.Unix(1_700_000_000, 0)

func sample(sec int, ms float64, lost bool) Sample {
	return Sample{
		At:   t0.Add(time.Duration(sec) * time.Second),
		RTT:  time.Duration(ms * float64(time.Millisecond)),
		Lost: lost,
		From: netip.MustParseAddr("192.0.2.1"),
	}
}

func TestSummaryLossLatencyJitter(t *testing.T) {
	var s Series
	for i, ms := range []float64{10, 14, 10, 0, 12} {
		s.Add(sample(i, ms, ms == 0))
	}
	got := s.Summary(t0, t0.Add(time.Minute))
	if got.Sent != 5 || got.Lost != 1 || got.Loss != 20 {
		t.Fatalf("sent/lost/loss = %d/%d/%v", got.Sent, got.Lost, got.Loss)
	}
	if got.Avg != 11.5 || got.Best != 10 || got.Worst != 14 {
		t.Fatalf("avg/best/worst = %v/%v/%v", got.Avg, got.Best, got.Worst)
	}
	if math.Abs(got.Jitter-10.0/3) > 1e-9 {
		t.Fatalf("jitter = %v, want 3.33", got.Jitter)
	}
}

func TestSummaryRespectsWindow(t *testing.T) {
	var s Series
	for i := 0; i < 10; i++ {
		s.Add(sample(i, float64(i), false))
	}
	got := s.Summary(t0.Add(5*time.Second), t0.Add(7*time.Second))
	if got.Sent != 2 || got.Avg != 5.5 {
		t.Fatalf("window summary = %+v", got)
	}
}

func TestAddDropsOldSamples(t *testing.T) {
	var s Series
	s.Add(sample(0, 1, false))
	s.Add(sample(int(Keep/time.Second)+5, 1, false))
	if len(s.samples) != 1 {
		t.Fatalf("kept %d samples, want 1", len(s.samples))
	}
}

func TestBuckets(t *testing.T) {
	var s Series
	for i := 0; i < 20; i++ {
		s.Add(sample(i, 5, i == 3))
	}
	b := s.Buckets(t0, t0.Add(20*time.Second), 10*time.Second)
	if len(b) != 2 || b[0].Sent != 10 || b[0].Lost != 1 || b[1].Lost != 0 {
		t.Fatalf("buckets = %+v", b)
	}
}
