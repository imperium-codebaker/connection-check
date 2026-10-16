// Package stats keeps recent probe samples and summarises them over a time window.
package stats

import (
	"math"
	"net/netip"
	"slices"
	"time"
)

// Keep is how long samples are retained.
const Keep = 15 * time.Minute

// Sample is one probe's outcome, stamped with its send time.
type Sample struct {
	At   time.Time
	RTT  time.Duration
	Lost bool
	From netip.Addr
}

// Summary describes the samples in one window. Times are in milliseconds, loss in percent.
type Summary struct {
	Sent, Lost             int
	Loss, Avg, Best, Worst float64
	Jitter                 float64 // mean difference between consecutive replies
	From                   netip.Addr
}

// Bucket summarises one slice of a timeline.
type Bucket struct {
	Start      time.Time
	Sent, Lost int
	Avg        float64
}

// Series holds recent samples for one hop or the destination.
type Series struct {
	samples []Sample
}

// Add records x and drops samples older than Keep.
func (s *Series) Add(x Sample) {
	s.samples = append(s.samples, x)
	cutoff := x.At.Add(-Keep)
	i := 0
	for i < len(s.samples) && s.samples[i].At.Before(cutoff) {
		i++
	}
	if i > 0 {
		s.samples = slices.Delete(s.samples, 0, i)
	}
}

// Summary summarises samples sent within [from, to).
func (s *Series) Summary(from, to time.Time) Summary {
	in := s.window(from, to)
	sum := Summary{Sent: len(in)}
	var replies []Sample
	for _, x := range in {
		if x.Lost {
			sum.Lost++
		} else {
			replies = append(replies, x)
		}
	}
	if sum.Sent > 0 {
		sum.Loss = 100 * float64(sum.Lost) / float64(sum.Sent)
	}
	fillLatency(&sum, replies)
	return sum
}

// Buckets splits [from, to) into slices of width and summarises each.
func (s *Series) Buckets(from, to time.Time, width time.Duration) []Bucket {
	var out []Bucket
	for start := from; start.Before(to); start = start.Add(width) {
		sum := s.Summary(start, start.Add(width))
		out = append(out, Bucket{Start: start, Sent: sum.Sent, Lost: sum.Lost, Avg: sum.Avg})
	}
	return out
}

func (s *Series) window(from, to time.Time) []Sample {
	var in []Sample
	for _, x := range s.samples {
		if !x.At.Before(from) && x.At.Before(to) {
			in = append(in, x)
		}
	}
	return in
}

// fillLatency sets the latency fields and most common responder from replies.
func fillLatency(sum *Summary, replies []Sample) {
	if len(replies) == 0 {
		return
	}
	slices.SortFunc(replies, func(a, b Sample) int { return a.At.Compare(b.At) })
	sum.Best = math.Inf(1)
	var total, diffs float64
	for i, x := range replies {
		ms := millis(x.RTT)
		total += ms
		sum.Best = math.Min(sum.Best, ms)
		sum.Worst = math.Max(sum.Worst, ms)
		if i > 0 {
			diffs += math.Abs(ms - millis(replies[i-1].RTT))
		}
	}
	sum.Avg = total / float64(len(replies))
	if len(replies) > 1 {
		sum.Jitter = diffs / float64(len(replies)-1)
	}
	sum.From = commonest(replies)
}

func commonest(replies []Sample) netip.Addr {
	counts := map[netip.Addr]int{}
	var best netip.Addr
	for _, x := range replies {
		counts[x.From]++
		if counts[x.From] > counts[best] {
			best = x.From
		}
	}
	return best
}

func millis(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
