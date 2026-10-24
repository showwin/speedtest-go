package speedtest

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// A proxy acknowledges upload bytes itself, so uploads through one count confirmed bytes.
func TestUploadThroughProxyCountsConfirmedBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
	}))
	defer server.Close()
	proxied := false
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxied = true
		r.RequestURI = ""
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	defer proxy.Close()

	client := New(WithUserConfig(&UserConfig{Proxy: proxy.URL}))
	target := &Server{URL: server.URL, Context: client}
	if err := uploadRequest(context.Background(), target, 0); err != nil {
		t.Fatal(err)
	}
	if !proxied {
		t.Fatal("the upload did not go through the proxy")
	}
	if _, counted := client.Manager.(*DataManager).upload.acked.total(); counted {
		t.Fatal("counted acknowledged bytes on a connection to the proxy")
	}
}

// Checks the platform's acknowledged-byte reading against the bytes a connection wrote,
// once the peer has read them all.
func TestConnAckedMatchesBytesWritten(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_, _ = io.Copy(io.Discard, conn)
		}
	}()
	// SPEEDTEST_ACK_PEER names a remote discard server, for platforms whose loopback differs
	// from a network path.
	peer := os.Getenv("SPEEDTEST_ACK_PEER")
	if peer == "" {
		peer = listener.Addr().String()
	}
	conn, err := countingDialer((&net.Dialer{}).DialContext)(context.Background(), "tcp", peer)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, ok := connAcked(conn); !ok {
		t.Skip("this platform does not report acknowledged bytes")
	}
	const size = 8 << 20
	if _, err := conn.Write(make([]byte, size)); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		acked, _ := connAcked(conn)
		if acked == size {
			return
		}
		if acked > size || time.Now().After(deadline) {
			t.Fatalf("acknowledged %d bytes of %d written", acked, size)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
