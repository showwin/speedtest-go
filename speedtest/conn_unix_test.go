//go:build linux || darwin

package speedtest

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// smallBufferListener fixes each accepted connection's receive buffer, so the server's
// acknowledgements follow its reads instead of filling a large autotuned buffer.
type smallBufferListener struct{ net.Listener }

func (l smallBufferListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetReadBuffer(32 * 1024)
	}
	return conn, err
}

// A server that reads uploads slowly never completes a request during the test. The upload
// rate must still be measured from acknowledged bytes rather than reported as zero, and match
// the rate the server read at.
func TestUploadMeasuresAcknowledgedBytesBeforeRequestsComplete(t *testing.T) {
	const rate = 500_000 // bytes a second, shared by all connections
	var mu sync.Mutex
	var reads []time.Time // one per 10 kB the server reads
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			return
		}
		buf := make([]byte, 10_000)
		for {
			mu.Lock()
			time.Sleep(time.Second * time.Duration(len(buf)) / rate)
			_, err := io.ReadFull(r.Body, buf)
			if err == nil {
				reads = append(reads, time.Now())
			}
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server.Listener = smallBufferListener{listener}
	server.Start()
	defer server.Close()

	client := New()
	client.SetCaptureTime(4 * time.Second)
	client.SetNThread(4)
	s := &Server{URL: server.URL + "/upload.php", ID: "1", Context: client}
	start := time.Now()
	if err := s.UploadTestContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	end := start.Add(*s.TestDuration.Upload)
	if confirmed := confirmedUpload(client); confirmed != 0 {
		t.Fatalf("%d bytes were confirmed; the test needs requests that do not complete", confirmed)
	}
	mu.Lock()
	defer mu.Unlock()
	// The meter measures the last two thirds of the test, so compare the server's reads there.
	from, inWindow, beforeEnd := start.Add(end.Sub(start)/3), 0, 0
	for _, at := range reads {
		if !at.After(end) {
			beforeEnd++
			if at.After(from) {
				inWindow++
			}
		}
	}
	read := float64(inWindow*10_000) / end.Sub(from).Seconds()
	if got := float64(s.ULSpeed); got < read*0.8 || got > read*1.2 {
		t.Fatalf("upload rate is %.0f bytes/s, and the server read %.0f bytes/s", got, read)
	}
	// The server's TCP stack has acknowledged every byte the server read during the test.
	if got, sent := client.GetTotalUpload(), int64(beforeEnd*10_000); got < sent {
		t.Fatalf("upload total is %d bytes, but the server read %d during the test", got, sent)
	}
}
