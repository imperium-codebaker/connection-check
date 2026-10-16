package probe

import (
	"encoding/binary"
	"errors"
	"net"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// udpPort is the destination port of UDP hop probes; it matches PCoIP's so probes follow the same links.
const udpPort = 4172

// reducedSender finds hops with UDP probes and times the destination with TCP connects.
type reducedSender struct {
	sock    *errSocket
	replies chan Reply
}

func openReduced() (Sender, error) {
	sock, err := openErrSocket(unix.IPPROTO_UDP)
	if err != nil {
		return nil, err
	}
	s := &reducedSender{sock: sock, replies: make(chan Reply, 256)}
	go sock.loop(s.handle)
	return s, nil
}

func (s *reducedSender) Send(p Packet) error {
	if p.TTL == DestTTL {
		go s.connect(p)
		return nil
	}
	return s.sock.send(binary.BigEndian.AppendUint16(nil, p.Seq), p.Dst, udpPort, p.TTL)
}

// connect times a TCP handshake; a refused connection still proves the host answered.
func (s *reducedSender) connect(p Packet) {
	addr := net.JoinHostPort(p.Dst.String(), strconv.Itoa(p.Port))
	c, err := net.DialTimeout("tcp4", addr, Timeout)
	at := time.Now()
	if err == nil {
		c.Close()
	}
	if err == nil || errors.Is(err, syscall.ECONNREFUSED) {
		s.replies <- Reply{Seq: p.Seq, Dst: p.Dst, From: p.Dst, At: at}
	}
}

// handle turns a router's error quoting our UDP probe into a Reply.
func (s *reducedSender) handle(in incoming) {
	if !in.offender.IsValid() || len(in.data) < 2 {
		return
	}
	seq := binary.BigEndian.Uint16(in.data[:2])
	s.replies <- Reply{Seq: seq, Dst: in.peer, From: in.offender, At: in.at, Unreachable: in.unreachable}
}

func (s *reducedSender) Replies() <-chan Reply { return s.replies }

func (s *reducedSender) Mode() Mode {
	return Mode{Name: "udp+tcp", Partial: true, DestPerSecond: 1, Hint: pingHint}
}

func (s *reducedSender) Close() error {
	s.sock.close()
	return nil
}
