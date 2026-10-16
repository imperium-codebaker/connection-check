package probe

import (
	"errors"

	"golang.org/x/sys/unix"
)

// pingHint tells Linux users how to let ordinary users send pings.
const pingHint = "This computer blocks pings from ordinary users, so some far hops may not answer. " +
	"For the full view, run once: echo 'net.ipv4.ping_group_range = 0 2147483647' | " +
	"sudo tee /etc/sysctl.d/60-ping-group.conf && sudo sysctl --system — then restart Connection Check."

// pingSender uses Linux's unprivileged ICMP datagram socket.
type pingSender struct {
	sock    *errSocket
	replies chan Reply
}

// Open returns the ICMP sender when ordinary users may ping, else the reduced UDP and TCP sender.
func Open() (Sender, error) {
	sock, err := openErrSocket(unix.IPPROTO_ICMP)
	if errors.Is(err, unix.EACCES) || errors.Is(err, unix.EPERM) {
		return openReduced()
	}
	if err != nil {
		return nil, err
	}
	s := &pingSender{sock: sock, replies: make(chan Reply, 256)}
	go sock.loop(s.handle)
	return s, nil
}

func (s *pingSender) Send(p Packet) error {
	return s.sock.send(echoRequest(0, p.Seq), p.Dst, 0, p.TTL)
}

// handle turns an echo reply, or a router's error quoting our echo request, into a Reply.
func (s *pingSender) handle(in incoming) {
	if in.offender.IsValid() {
		if seq, ok := echoSeq(in.data, 8); ok {
			s.replies <- Reply{Seq: seq, Dst: in.peer, From: in.offender, At: in.at}
		}
		return
	}
	if seq, ok := echoSeq(in.data, 0); ok {
		s.replies <- Reply{Seq: seq, Dst: in.peer, From: in.peer, At: in.at}
	}
}

func (s *pingSender) Replies() <-chan Reply { return s.replies }

func (s *pingSender) Mode() Mode { return Mode{Name: "icmp", DestPerSecond: 5} }

func (s *pingSender) Close() error {
	s.sock.close()
	return nil
}
