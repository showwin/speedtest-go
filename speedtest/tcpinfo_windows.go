package speedtest

import (
	"net"
	"unsafe"

	"golang.org/x/sys/windows"
)

// tcpInfoV0 is TCP_INFO_v0 from mstcpip.h.
type tcpInfoV0 struct {
	State             uint32
	Mss               uint32
	ConnectionTimeMs  uint64
	TimestampsEnabled bool
	RttUs             uint32
	MinRttUs          uint32
	BytesInFlight     uint32
	Cwnd              uint32
	SndWnd            uint32
	RcvWnd            uint32
	RcvBuf            uint32
	BytesOut          uint64
	BytesIn           uint64
	BytesReordered    uint32
	BytesRetrans      uint32
	FastRetrans       uint32
	DupAcksIn         uint32
	TimeoutEpisodes   uint32
	SynRetrans        uint8
}

const sioTCPInfo = 0xD8000027 // SIO_TCP_INFO: _WSAIORW(IOC_VENDOR, 39)

func tcpAcked(conn *net.TCPConn, _ int64) (acked int64, ok bool) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, false
	}
	_ = raw.Control(func(fd uintptr) {
		var version uint32
		var info tcpInfoV0
		var returned uint32
		err := windows.WSAIoctl(windows.Handle(fd), sioTCPInfo, (*byte)(unsafe.Pointer(&version)), uint32(unsafe.Sizeof(version)),
			(*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), &returned, nil, 0)
		if err == nil {
			acked, ok = int64(info.BytesOut)-int64(info.BytesRetrans)-int64(info.BytesInFlight), true
		}
	})
	return acked, ok
}
