package probe

import (
	"encoding/binary"
	"net/netip"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	iphlpapi        = windows.NewLazySystemDLL("iphlpapi.dll")
	icmpCreateFile  = iphlpapi.NewProc("IcmpCreateFile")
	icmpCloseHandle = iphlpapi.NewProc("IcmpCloseHandle")
	icmpSendEcho2   = iphlpapi.NewProc("IcmpSendEcho2")
)

// Statuses from ipexport.h that carry a responding address.
const (
	ipSuccess           = 0
	ipDestNetUnreach    = 11002
	ipDestHostUnreach   = 11003
	ipDestProtUnreach   = 11004
	ipDestPortUnreach   = 11005
	ipTTLExpiredTransit = 11013
)

// ipOptionInformation mirrors IP_OPTION_INFORMATION.
type ipOptionInformation struct {
	TTL         uint8
	Tos         uint8
	Flags       uint8
	OptionsSize uint8
	OptionsData uintptr
}

// icmpEchoReply mirrors ICMP_ECHO_REPLY.
type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	Options       ipOptionInformation
}

// echoSender uses the Windows ICMP helper API, which needs no administrator rights.
type echoSender struct {
	replies chan Reply
}

// Open returns the Sender for this system.
func Open() (Sender, error) {
	if err := icmpSendEcho2.Find(); err != nil {
		return nil, err
	}
	return &echoSender{replies: make(chan Reply, 256)}, nil
}

// Send runs one blocking echo in the background; the API waits up to Timeout for a reply.
func (s *echoSender) Send(p Packet) error {
	go s.echo(p)
	return nil
}

func (s *echoSender) echo(p Packet) {
	h, _, _ := icmpCreateFile.Call()
	if windows.Handle(h) == windows.InvalidHandle {
		return
	}
	defer icmpCloseHandle.Call(h)
	if from, ok := sendEcho(h, p); ok {
		s.replies <- Reply{Seq: p.Seq, Dst: p.Dst, From: from, At: time.Now()}
	}
}

// sendEcho sends one echo request with p's hop limit and returns the address that answered.
func sendEcho(h uintptr, p Packet) (netip.Addr, bool) {
	dst := p.Dst.As4()
	opts := ipOptionInformation{TTL: uint8(p.TTL)}
	buf := make([]byte, 1024)
	icmpSendEcho2.Call(h, 0, 0, 0,
		uintptr(binary.LittleEndian.Uint32(dst[:])),
		uintptr(unsafe.Pointer(&payload[0])), uintptr(len(payload)),
		uintptr(unsafe.Pointer(&opts)),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)),
		uintptr(Timeout.Milliseconds()))
	reply := (*icmpEchoReply)(unsafe.Pointer(&buf[0]))
	if reply.Address == 0 || !answered(reply.Status) {
		return netip.Addr{}, false
	}
	var a [4]byte
	binary.LittleEndian.PutUint32(a[:], reply.Address)
	return netip.AddrFrom4(a), true
}

func answered(status uint32) bool {
	switch status {
	case ipSuccess, ipTTLExpiredTransit, ipDestNetUnreach, ipDestHostUnreach, ipDestProtUnreach, ipDestPortUnreach:
		return true
	}
	return false
}

func (s *echoSender) Replies() <-chan Reply { return s.replies }

func (s *echoSender) Mode() Mode { return Mode{Name: "icmp", DestPerSecond: 5} }

func (s *echoSender) Close() error { return nil }
