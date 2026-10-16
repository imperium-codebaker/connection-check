// Package probe sends hop-limited pings and matches their replies.
package probe

import (
	"net/netip"
	"time"
)

// DestTTL is the hop limit used for probes meant to reach the destination.
const DestTTL = 64

// MaxTTL is the deepest hop probed while the route length is unknown.
const MaxTTL = 30

// Timeout is how long a probe may wait for its reply before counting as lost.
const Timeout = 2 * time.Second

// Packet is one probe to send.
type Packet struct {
	Dst  netip.Addr
	TTL  int
	Seq  uint16
	Port int // TCP port, used only by senders that time TCP connects
}

// Reply is a response matched to a sent packet by sequence number.
type Reply struct {
	Seq  uint16
	Dst  netip.Addr // the probe's original destination
	From netip.Addr // the router or host that answered
	At   time.Time
	// Unreachable is true when the answer was a destination-unreachable error rather than
	// an echo reply or an expired hop limit.
	Unreachable bool
}

// Mode describes what a Sender can measure.
type Mode struct {
	Name          string
	Partial       bool
	DestPerSecond int
	Hint          string
}

// Sender sends probes and delivers replies.
type Sender interface {
	Send(p Packet) error
	Replies() <-chan Reply
	Mode() Mode
	Close() error
}

// Result is the outcome of one probe.
type Result struct {
	Dst  netip.Addr
	TTL  int
	Sent time.Time
	From netip.Addr // zero when lost
	RTT  time.Duration
	Lost bool
}
