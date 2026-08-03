//go:build !linux

package speedtest

import (
	"net"
	"syscall"
)

func resolveInterface(_ string) (net.IP, func(string, string, syscall.RawConn) error, bool) {
	return nil, nil, false
}
