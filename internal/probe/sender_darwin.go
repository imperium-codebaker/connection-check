package probe

import (
	"net"
	"net/netip"
	"os"
	"sync"
	"time"

	"golang.org/x/net/icmp"
)

// dgramSender uses macOS's unprivileged ICMP datagram socket.
type dgramSender struct {
	c       *icmp.PacketConn
	id      int
	mu      sync.Mutex
	replies chan Reply
}

// Open returns the Sender for this system.
func Open() (Sender, error) {
	c, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return nil, err
	}
	s := &dgramSender{c: c, id: os.Getpid() & 0xffff, replies: make(chan Reply, 256)}
	go s.read()
	return s, nil
}

func (s *dgramSender) Send(p Packet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.c.IPv4PacketConn().SetTTL(p.TTL); err != nil {
		return err
	}
	_, err := s.c.WriteTo(echoRequest(s.id, p.Seq), &net.UDPAddr{IP: p.Dst.AsSlice()})
	return err
}

func (s *dgramSender) read() {
	buf := make([]byte, 1500)
	for {
		n, peer, err := s.c.ReadFrom(buf)
		if err != nil {
			return
		}
		at := time.Now()
		from, _ := netip.AddrFromSlice(peer.(*net.UDPAddr).IP.To4())
		if r, ok := parseICMP(buf[:n], from); ok {
			r.At = at
			s.replies <- r
		}
	}
}

func (s *dgramSender) Replies() <-chan Reply { return s.replies }

func (s *dgramSender) Mode() Mode { return Mode{Name: "icmp", DestPerSecond: 5} }

func (s *dgramSender) Close() error { return s.c.Close() }
