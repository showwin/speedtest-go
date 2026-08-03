//go:build linux

package speedtest

import (
	"net"
	"syscall"
)

// resolveInterface checks if source is a network interface name.
// If so, it returns the interface's first IPv4 address and a
// DialerControl function that binds sockets to the interface
// via SO_BINDTODEVICE.
func resolveInterface(source string) (ip net.IP, control func(string, string, syscall.RawConn) error, ok bool) {
	iface, err := net.InterfaceByName(source)
	if err != nil {
		return nil, nil, false
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, nil, false
	}
	for _, a := range addrs {
		ipnet, isIPNet := a.(*net.IPNet)
		if isIPNet && ipnet.IP.To4() != nil {
			devName := iface.Name
			ctrl := func(_, _ string, c syscall.RawConn) error {
				var serr error
				if err := c.Control(func(fd uintptr) {
					serr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, devName)
				}); err != nil {
					return err
				}
				return serr
			}
			return ipnet.IP, ctrl, true
		}
	}
	return nil, nil, false
}
