package probe

import (
	"net/netip"
	"slices"
	"testing"
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
