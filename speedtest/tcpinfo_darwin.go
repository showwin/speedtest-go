package speedtest

import (
	"net"

	"golang.org/x/sys/unix"
)

// macOS reports the bytes held in the send buffer, which keeps data until it is acknowledged,
// so the acknowledged bytes are the bytes written less that.
func tcpAcked(conn *net.TCPConn, written int64) (acked int64, ok bool) {
	if written < 0 {
		return 0, false
	}
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, false
	}
	_ = raw.Control(func(fd uintptr) {
		info, err := unix.GetsockoptTCPConnectionInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_CONNECTION_INFO)
		if err == nil {
			acked, ok = written-int64(info.Snd_sbbytes), true
		}
	})
	return acked, ok
}
