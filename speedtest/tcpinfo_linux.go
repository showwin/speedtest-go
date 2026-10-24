package speedtest

import (
	"net"

	"golang.org/x/sys/unix"
)

func tcpAcked(conn *net.TCPConn, _ int64) (acked int64, ok bool) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, false
	}
	_ = raw.Control(func(fd uintptr) {
		info, err := unix.GetsockoptTCPInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_INFO)
		// Linux counts the SYN, so a connection reports at least 1; kernels before 4.1 report 0.
		if err == nil && info.Bytes_acked > 0 {
			acked, ok = int64(info.Bytes_acked)-1, true
		}
	})
	return acked, ok
}
