package transport

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestClientDownload(t *testing.T) {
	const size int64 = 32
	want := bytes.Repeat([]byte("JABCDEFGHI"), 4)[:size]
	addr, stop := startTestTCPServer(t, func(conn net.Conn) {
		reader := bufio.NewReader(conn)
		if line, _ := reader.ReadString('\n'); strings.TrimSpace(line) != "HI" {
			t.Errorf("unexpected handshake: %q", line)
		}
		_, _ = conn.Write([]byte("HELLO 2.1 test\n"))
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(line) != "DOWNLOAD 32" {
			t.Errorf("unexpected download command: %q", line)
		}
		_, _ = conn.Write(want[:7])
		_, _ = conn.Write(want[7:])
	})
	defer stop()

	client, err := NewClient(&net.Dialer{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err = client.Connect(context.Background(), addr); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Disconnect() }()
	if _, err := client.VersionContext(context.Background()); err != nil {
		t.Fatal("handshake failed")
	}
	var got bytes.Buffer
	err = client.Download(context.Background(), size, func(r io.Reader) error {
		_, err := io.Copy(&got, r)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("download data mismatch: got %q want %q", got.Bytes(), want)
	}
}

func TestClientUploadUsesDeclaredTotal(t *testing.T) {
	const size int64 = 20
	addr, stop := startTestTCPServer(t, func(conn net.Conn) {
		reader := bufio.NewReader(conn)
		_, _ = reader.ReadString('\n')
		_, _ = conn.Write([]byte("HELLO 2.1 test\n"))
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(line) != "UPLOAD 20 0" {
			t.Errorf("unexpected upload command: %q", line)
		}
		payloadSize := size - int64(len(line)) - 1
		payload := make([]byte, payloadSize+1)
		if _, err := io.ReadFull(reader, payload); err != nil {
			t.Errorf("read upload payload: %v", err)
		}
		if payload[len(payload)-1] != '\n' {
			t.Errorf("upload payload is missing final newline")
		}
		_, _ = conn.Write([]byte("OK 20 0\n"))
	})
	defer stop()

	client, err := NewClient(&net.Dialer{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err = client.Connect(context.Background(), addr); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Disconnect() }()
	if _, err := client.VersionContext(context.Background()); err != nil {
		t.Fatal("handshake failed")
	}
	payloadSize, err := UploadPayloadSize(size)
	if err != nil || payloadSize != 7 {
		t.Fatalf("payload size = %d, want 7 (err=%v)", payloadSize, err)
	}
	acknowledged, err := client.Upload(context.Background(), size, bytes.NewReader(bytes.Repeat([]byte("x"), int(size))))
	if err != nil {
		t.Fatal(err)
	}
	if acknowledged != size {
		t.Fatalf("acknowledged %d bytes, want %d", acknowledged, size)
	}
}

func startTestTCPServer(t *testing.T, handler func(net.Conn)) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-stop:
					return
				default:
					t.Error(err)
					return
				}
			}
			go func() {
				defer func() { _ = conn.Close() }()
				handler(conn)
			}()
		}
	}()
	return listener.Addr().String(), func() {
		close(stop)
		_ = listener.Close()
	}
}
