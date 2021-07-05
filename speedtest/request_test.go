package speedtest

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDownloadTestContext(t *testing.T) {
	latency, _ := time.ParseDuration("5ms")
	server := Server{
		URL:     "http://dummy.com/upload.php",
		Latency: latency,
	}

	err := server.DownloadTestContext(
		context.Background(),
		false,
		mockWarmUp,
		mockRequest,
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	if server.DLSpeed < 6000 || 6300 < server.DLSpeed {
		t.Errorf("got unexpected server.DLSpeed '%v', expected between 6000 and 6300", server.DLSpeed)
	}
}

func TestDownloadTestContextSavingMode(t *testing.T) {
	latency, _ := time.ParseDuration("5ms")
	server := Server{
		URL:     "http://dummy.com/upload.php",
		Latency: latency,
	}

	err := server.DownloadTestContext(
		context.Background(),
		true,
		mockWarmUp,
		mockRequest,
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	if server.DLSpeed < 180 || 200 < server.DLSpeed {
		t.Errorf("got unexpected server.DLSpeed '%v', expected between 180 and 200", server.DLSpeed)
	}
}

func TestUploadTestContext(t *testing.T) {
	latency, _ := time.ParseDuration("5ms")
	server := Server{
		URL:     "http://dummy.com/upload.php",
		Latency: latency,
	}

	err := server.UploadTestContext(
		context.Background(),
		false,
		mockWarmUp,
		mockRequest,
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	if server.ULSpeed < 2400 || 2600 < server.ULSpeed {
		t.Errorf("got unexpected server.ULSpeed '%v', expected between 2400 and 2600", server.ULSpeed)
	}
}

func TestUploadTestContextSavingMode(t *testing.T) {
	latency, _ := time.ParseDuration("5ms")
	server := Server{
		URL:     "http://dummy.com/upload.php",
		Latency: latency,
	}

	err := server.UploadTestContext(
		context.Background(),
		true,
		mockWarmUp,
		mockRequest,
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	if server.ULSpeed < 45 || 50 < server.ULSpeed {
		t.Errorf("got unexpected server.ULSpeed '%v', expected between 45 and 50", server.ULSpeed)
	}
}

func mockWarmUp(ctx context.Context, dlURL string) error {
	time.Sleep(100 * time.Millisecond)
	return nil
}

func mockRequest(ctx context.Context, dlURL string, w int) error {
	fmt.Sprintln(w)
	time.Sleep(500 * time.Millisecond)
	return nil
}
