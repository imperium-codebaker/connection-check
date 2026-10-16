package probe

import (
	"encoding/binary"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// errSocket is a datagram socket with IP_RECVERR, so routers' ICMP errors arrive on its error queue.
type errSocket struct {
	fd     int
	mu     sync.Mutex
	closed atomic.Bool
}

// incoming is one datagram or error-queue entry read from an errSocket.
type incoming struct {
	data     []byte
	peer     netip.Addr // sender for normal data; original destination for errors
	offender netip.Addr // router that sent the error; zero for normal data
	// unreachable is true when the error is destination-unreachable rather than an expired hop limit.
	unreachable bool
	at          time.Time
}

func openErrSocket(proto int) (*errSocket, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, proto)
	if err != nil {
		return nil, err
	}
	if err := unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_RECVERR, 1); err != nil {
		unix.Close(fd)
		return nil, err
	}
	return &errSocket{fd: fd}, nil
}

// send writes b to dst:port with the given hop limit.
func (s *errSocket) send(b []byte, dst netip.Addr, port, ttl int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := unix.SetsockoptInt(s.fd, unix.IPPROTO_IP, unix.IP_TTL, ttl); err != nil {
		return err
	}
	return unix.Sendto(s.fd, b, 0, &unix.SockaddrInet4{Addr: dst.As4(), Port: port})
}

// loop reads datagrams and error-queue entries and hands each to handle until close.
func (s *errSocket) loop(handle func(incoming)) {
	defer unix.Close(s.fd)
	for !s.closed.Load() {
		pfd := []unix.PollFd{{Fd: int32(s.fd), Events: unix.POLLIN}}
		if n, _ := unix.Poll(pfd, 250); n <= 0 {
			continue
		}
		if pfd[0].Revents&unix.POLLERR != 0 {
			s.readErr(handle)
		}
		if pfd[0].Revents&unix.POLLIN != 0 {
			s.readData(handle)
		}
	}
}

func (s *errSocket) close() { s.closed.Store(true) }

func (s *errSocket) readData(handle func(incoming)) {
	buf := make([]byte, 1500)
	n, from, err := unix.Recvfrom(s.fd, buf, unix.MSG_DONTWAIT)
	if sa, ok := from.(*unix.SockaddrInet4); err == nil && ok {
		handle(incoming{data: buf[:n], peer: netip.AddrFrom4(sa.Addr), at: time.Now()})
	}
}

func (s *errSocket) readErr(handle func(incoming)) {
	buf, oob := make([]byte, 1500), make([]byte, 512)
	n, oobn, _, from, err := unix.Recvmsg(s.fd, buf, oob, unix.MSG_ERRQUEUE|unix.MSG_DONTWAIT)
	sa, ok := from.(*unix.SockaddrInet4)
	if err != nil || !ok {
		return
	}
	if off, unreachable, ok := offender(oob[:oobn]); ok {
		handle(incoming{data: buf[:n], peer: netip.AddrFrom4(sa.Addr), offender: off,
			unreachable: unreachable, at: time.Now()})
	}
}

// offender returns the address of the router named in an IP_RECVERR control message, and
// whether its error was destination-unreachable.
func offender(oob []byte) (addr netip.Addr, unreachable, ok bool) {
	msgs, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return netip.Addr{}, false, false
	}
	size := int(unsafe.Sizeof(unix.SockExtendedErr{}))
	for _, m := range msgs {
		if m.Header.Level != unix.IPPROTO_IP || m.Header.Type != unix.IP_RECVERR || len(m.Data) < size {
			continue
		}
		ee := (*unix.SockExtendedErr)(unsafe.Pointer(&m.Data[0]))
		off := m.Data[size:]
		if len(off) >= 8 && binary.NativeEndian.Uint16(off[0:2]) == unix.AF_INET {
			return netip.AddrFrom4([4]byte(off[4:8])), ee.Type == icmpDestUnreach, true
		}
	}
	return netip.Addr{}, false, false
}

// icmpDestUnreach is the ICMPv4 destination-unreachable message type.
const icmpDestUnreach = 3
