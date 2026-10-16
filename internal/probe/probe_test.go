package probe

import (
	"net/netip"
	"slices"
	"testing"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func TestScheduleSpreadsDestinationProbes(t *testing.T) {
	got := schedule(4, 2)
	want := []int{DestTTL, 1, 2, DestTTL, 3, 4}
	if !slices.Equal(got, want) {
		t.Fatalf("schedule(4, 2) = %v, want %v", got, want)
	}
	if n := len(schedule(MaxTTL, 5)); n != MaxTTL+5 {
		t.Fatalf("schedule(MaxTTL, 5) has %d slots", n)
	}
}

func TestQuotedProbeReadsDestinationAndSequence(t *testing.T) {
	ip := []byte{0x45, 0, 0, 44, 0, 0, 0, 0, 1, 1, 0, 0, 10, 0, 0, 2, 192, 0, 2, 47}
	quoted := append(ip, echoRequest(7, 513)[:8]...)
	dst, seq, ok := quotedProbe(quoted)
	if !ok || seq != 513 || dst != netip.MustParseAddr("192.0.2.47") {
		t.Fatalf("quotedProbe = %v %d %v", dst, seq, ok)
	}
}

func TestQuotedProbeRejectsOtherProtocols(t *testing.T) {
	ip := []byte{0x45, 0, 0, 44, 0, 0, 0, 0, 1, 17, 0, 0, 10, 0, 0, 2, 192, 0, 2, 47}
	if _, _, ok := quotedProbe(append(ip, make([]byte, 8)...)); ok {
		t.Fatal("quoted UDP header accepted as an echo request")
	}
}

func TestParseICMPEchoReply(t *testing.T) {
	b := echoRequest(1, 42)
	b[0] = 0
	peer := netip.MustParseAddr("192.0.2.47")
	r, ok := parseICMP(b, peer)
	if !ok || r.Seq != 42 || r.From != peer || r.Dst != peer {
		t.Fatalf("parseICMP = %+v %v", r, ok)
	}
}

// icmpError returns an ICMP error of type typ from a router, quoting an echo request to dst.
func icmpError(t *testing.T, typ ipv4.ICMPType, dst netip.Addr, seq uint16) []byte {
	t.Helper()
	ip := append([]byte{0x45, 0, 0, 44, 0, 0, 0, 0, 1, 1, 0, 0, 10, 0, 0, 2}, dst.AsSlice()...)
	quoted := append(ip, echoRequest(7, seq)[:8]...)
	var body icmp.MessageBody = &icmp.TimeExceeded{Data: quoted}
	if typ == ipv4.ICMPTypeDestinationUnreachable {
		body = &icmp.DstUnreach{Data: quoted}
	}
	b, err := (&icmp.Message{Type: typ, Code: 1, Body: body}).Marshal(nil)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestOnlyRealAnswersCountAsReplies(t *testing.T) {
	dst, router := netip.MustParseAddr("192.0.2.47"), netip.MustParseAddr("192.0.2.1")
	echo := echoRequest(1, 9)
	echo[0] = 0
	cases := []struct {
		name     string
		ttl      int
		msg      []byte
		peer     netip.Addr
		wantLost bool
	}{
		{"router expires a hop probe", 3, icmpError(t, ipv4.ICMPTypeTimeExceeded, dst, 9), router, false},
		{"destination answers its probe", DestTTL, echo, dst, false},
		{"destination refuses a hop probe that reached it", 9, icmpError(t, ipv4.ICMPTypeDestinationUnreachable, dst, 9), dst, false},
		{"router refuses a destination probe", DestTTL, icmpError(t, ipv4.ICMPTypeDestinationUnreachable, dst, 9), router, true},
		{"router refuses a hop probe", 3, icmpError(t, ipv4.ICMPTypeDestinationUnreachable, dst, 9), router, true},
		{"router expires a destination probe", DestTTL, icmpError(t, ipv4.ICMPTypeTimeExceeded, dst, 9), router, true},
	}
	for _, c := range cases {
		r, ok := parseICMP(c.msg, c.peer)
		if !ok {
			t.Fatalf("%s: not parsed", c.name)
		}
		var got []Result
		e := NewEngine(nil, func(x Result) { got = append(got, x) })
		sent := time.Now()
		e.pending[9] = pending{dst: dst, ttl: c.ttl, sent: sent}
		r.At = sent.Add(time.Millisecond)
		e.match(r)
		e.expire(sent.Add(2 * Timeout))
		if len(got) != 1 || got[0].Lost != c.wantLost {
			t.Errorf("%s: got %+v, want lost=%v", c.name, got, c.wantLost)
		}
	}
}
