package probe

import (
	"encoding/binary"
	"net/netip"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// payload is the data carried by every echo request.
var payload = []byte("connection-check")

// echoRequest returns an ICMPv4 echo request with the given identifier and sequence number.
func echoRequest(id int, seq uint16) []byte {
	m := icmp.Message{Type: ipv4.ICMPTypeEcho, Body: &icmp.Echo{ID: id, Seq: int(seq), Data: payload}}
	b, _ := m.Marshal(nil)
	return b
}

// echoSeq returns the sequence number of an ICMP echo request or reply that starts at b.
func echoSeq(b []byte, wantType byte) (uint16, bool) {
	if len(b) < 8 || b[0] != wantType {
		return 0, false
	}
	return binary.BigEndian.Uint16(b[6:8]), true
}

// quotedProbe returns the destination and sequence number of the echo request
// quoted inside an ICMP error (original IP header followed by its first 8 bytes).
func quotedProbe(data []byte) (netip.Addr, uint16, bool) {
	if len(data) < 20 || data[0]>>4 != 4 {
		return netip.Addr{}, 0, false
	}
	ihl := int(data[0]&0x0f) * 4
	if len(data) < ihl+8 || data[9] != 1 {
		return netip.Addr{}, 0, false
	}
	seq, ok := echoSeq(data[ihl:], 8)
	return netip.AddrFrom4([4]byte(data[16:20])), seq, ok
}

// parseICMP turns a received ICMP message from peer into a Reply.
func parseICMP(b []byte, peer netip.Addr) (Reply, bool) {
	m, err := icmp.ParseMessage(1, b)
	if err != nil {
		return Reply{}, false
	}
	switch body := m.Body.(type) {
	case *icmp.Echo:
		if m.Type != ipv4.ICMPTypeEchoReply {
			return Reply{}, false
		}
		return Reply{Seq: uint16(body.Seq), Dst: peer, From: peer}, true
	case *icmp.TimeExceeded:
		return quotedReply(body.Data, peer)
	case *icmp.DstUnreach:
		return quotedReply(body.Data, peer)
	}
	return Reply{}, false
}

func quotedReply(data []byte, peer netip.Addr) (Reply, bool) {
	dst, seq, ok := quotedProbe(data)
	return Reply{Seq: seq, Dst: dst, From: peer}, ok
}
