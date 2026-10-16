package probe

import (
	"encoding/binary"
	"net/netip"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

// recvErr builds an IP_RECVERR control message for an ICMP error of type typ sent by router.
func recvErr(typ uint8, router netip.Addr) []byte {
	ee := unix.SockExtendedErr{Origin: unix.SO_EE_ORIGIN_ICMP, Type: typ}
	data := unsafe.Slice((*byte)(unsafe.Pointer(&ee)), unsafe.Sizeof(ee))
	sa := make([]byte, 16)
	binary.NativeEndian.PutUint16(sa[0:2], unix.AF_INET)
	copy(sa[4:8], router.AsSlice())
	data = append(append([]byte(nil), data...), sa...)
	b := make([]byte, unix.CmsgSpace(len(data)))
	h := (*unix.Cmsghdr)(unsafe.Pointer(&b[0]))
	h.Level, h.Type = unix.IPPROTO_IP, unix.IP_RECVERR
	h.SetLen(unix.CmsgLen(len(data)))
	copy(b[unix.CmsgLen(0):], data)
	return b
}

func TestOffenderReadsRouterAndErrorType(t *testing.T) {
	router := netip.MustParseAddr("192.0.2.1")
	for typ, wantUnreachable := range map[uint8]bool{11: false, icmpDestUnreach: true} {
		addr, unreachable, ok := offender(recvErr(typ, router))
		if !ok || addr != router || unreachable != wantUnreachable {
			t.Errorf("type %d: got %v unreachable=%v ok=%v", typ, addr, unreachable, ok)
		}
	}
}
