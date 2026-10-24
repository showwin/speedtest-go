//go:build !linux && !darwin && !windows

package speedtest

import "net"

func tcpAcked(*net.TCPConn, int64) (int64, bool) { return 0, false }
