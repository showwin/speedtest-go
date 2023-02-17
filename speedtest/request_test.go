package speedtest

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestDownloadTestContext(t *testing.T) {
	GlobalDataManager.Reset()

	idealSpeed := 0.1 * 8 * float64(runtime.NumCPU()) * 10 / 0.1 // one mockRequest per second with all CPU cores
	delta := 0.05

	latency, _ := time.ParseDuration("5ms")
	server := Server{
		URL:     "http://dummy.com/upload.php",
		Latency: latency,
		context: defaultClient,
	}

	err := server.downloadTestContext(
		context.Background(),
		false,
		mockWarmUp,
		mockRequest,
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	if server.DLSpeed < idealSpeed*(1-delta) || idealSpeed*(1+delta) < server.DLSpeed {
		t.Errorf("got unexpected server.DLSpeed '%v', expected between %v and %v", server.DLSpeed, idealSpeed*(1-delta), idealSpeed*(1+delta))
	}
}

func TestUploadTestContext(t *testing.T) {
	GlobalDataManager.Reset()

	idealSpeed := 0.1 * 8 * float64(runtime.NumCPU()) * 10 / 0.1 // one mockRequest per second with all CPU cores
	delta := 0.05                                                // tolerance scope (-0.05, +0.05)

	latency, _ := time.ParseDuration("5ms")
	server := Server{
		URL:     "http://dummy.com/upload.php",
		Latency: latency,
		context: defaultClient,
	}

	err := server.uploadTestContext(
		context.Background(),
		false,
		mockWarmUp,
		mockRequest,
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	if server.ULSpeed < idealSpeed*(1-delta) || idealSpeed*(1+delta) < server.ULSpeed {
		t.Errorf("got unexpected server.ULSpeed '%v', expected between %v and %v", server.ULSpeed, idealSpeed*(1-delta), idealSpeed*(1+delta))
	}
}

func mockWarmUp(ctx context.Context, doer *http.Client, dlURL string) error {
	time.Sleep(5000 * time.Millisecond)
	return nil
}

func mockRequest(ctx context.Context, doer *http.Client, dlURL string, w int) error {
	fmt.Sprintln(w)
	dc := GlobalDataManager.NewDataChunk()
	// (0.1MegaByte * 8bit * 8CPU * 10loop) / 0.1s = 640Megabit
	for i := 0; i < 10; i++ {
		atomic.AddInt64(&dc.manager.totalDownload, 1*1000*100)
		atomic.AddInt64(&dc.manager.totalUpload, 1*1000*100)
		time.Sleep(time.Millisecond * 10)
	}
	return nil
}
