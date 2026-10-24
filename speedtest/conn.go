package speedtest

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
)

// countedConn counts the bytes written to a connection, so its acknowledged bytes can be
// derived where the platform reports only the bytes still unacknowledged, and keeps its
// last reading once closed.
type countedConn struct {
	net.Conn
	address string // the address the transport dialed
	written atomic.Int64
	once    sync.Once
	final   atomic.Int64 // acknowledged bytes at close, or -1 while open
}

func countingDialer(dial func(ctx context.Context, network, address string) (net.Conn, error)) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		c := &countedConn{Conn: conn, address: address}
		c.final.Store(-1)
		return c, nil
	}
}

func (c *countedConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.written.Add(int64(n))
	return n, err
}

func (c *countedConn) Close() error {
	c.once.Do(func() {
		if acked, ok := c.acked(); ok {
			c.final.Store(acked)
		}
	})
	return c.Conn.Close()
}

func (c *countedConn) acked() (int64, bool) {
	if final := c.final.Load(); final >= 0 {
		return final, true
	}
	tcp, ok := c.Conn.(*net.TCPConn)
	if !ok {
		return 0, false
	}
	return tcpAcked(tcp, c.written.Load())
}

// dialedTo reports whether the client's transport dialed a connection to a URL's own host
// rather than to a proxy, so that the bytes acknowledged on it are the server's.
func dialedTo(conn net.Conn, u *url.URL) bool {
	if tlsConn, ok := conn.(*tls.Conn); ok {
		conn = tlsConn.NetConn()
	}
	c, ok := conn.(*countedConn)
	if !ok {
		return false
	}
	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}
	return c.address == net.JoinHostPort(u.Hostname(), port)
}

// connAcked returns the bytes the peer has acknowledged on a connection, when the platform
// reports them. `written` is unknown (-1) for a connection that was not dialed through
// countingDialer.
func connAcked(conn net.Conn) (int64, bool) {
	if tlsConn, ok := conn.(*tls.Conn); ok {
		conn = tlsConn.NetConn()
	}
	switch c := conn.(type) {
	case *countedConn:
		return c.acked()
	case *net.TCPConn:
		return tcpAcked(c, -1)
	}
	return 0, false
}

// ackedBytes sums the bytes acknowledged on the connections a test direction has used,
// counting each from the moment the direction first used it.
type ackedBytes struct {
	mu    sync.Mutex
	conns map[net.Conn]int64 // connection to its acknowledged bytes when first used
	ended int64              // bytes acknowledged on connections released before closing
}

// uploadAcked returns the acknowledged-byte counter of a client's upload test, or nil when
// the client has a custom Manager.
func uploadAcked(s *Speedtest) *ackedBytes {
	if dm, ok := s.Manager.(*DataManager); ok {
		return &dm.upload.acked
	}
	return nil
}

func (a *ackedBytes) use(conn net.Conn) {
	if a == nil {
		return
	}
	acked, ok := connAcked(conn)
	if !ok {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conns == nil {
		a.conns = map[net.Conn]int64{}
	}
	if _, seen := a.conns[conn]; !seen {
		a.conns[conn] = acked
	}
}

// release counts a connection's final bytes before it is closed by a path that bypasses
// countedConn, such as the TCP test protocol.
func (a *ackedBytes) release(conn net.Conn) {
	if a == nil {
		return
	}
	acked, ok := connAcked(conn)
	a.mu.Lock()
	defer a.mu.Unlock()
	if base, seen := a.conns[conn]; seen {
		if ok {
			a.ended += acked - base
		}
		delete(a.conns, conn)
	}
}

// total returns the acknowledged bytes, and false when no connection reports them.
func (a *ackedBytes) total() (int64, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	total, ok := a.ended, a.ended > 0
	for conn, base := range a.conns {
		if acked, reported := connAcked(conn); reported {
			total += acked - base
			ok = true
		}
	}
	return total, ok
}
