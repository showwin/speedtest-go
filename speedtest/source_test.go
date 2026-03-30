package speedtest

import (
	"net"
	"runtime"
	"testing"
)

func TestResolveInterface(t *testing.T) {
	t.Run("InvalidName", func(t *testing.T) {
		_, _, ok := resolveInterface("nonexistent0")
		if ok {
			t.Error("expected ok=false for nonexistent interface")
		}
	})

	t.Run("IPAddress", func(t *testing.T) {
		_, _, ok := resolveInterface("127.0.0.1")
		if ok {
			t.Error("expected ok=false for IP address")
		}
	})

	t.Run("Loopback", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("SO_BINDTODEVICE requires Linux")
		}
		ip, ctrl, ok := resolveInterface("lo")
		if !ok {
			t.Fatal("expected ok=true for lo")
		}
		if !ip.Equal(net.IPv4(127, 0, 0, 1)) {
			t.Errorf("expected 127.0.0.1, got %s", ip)
		}
		if ctrl == nil {
			t.Error("expected non-nil control function")
		}
	})
}
