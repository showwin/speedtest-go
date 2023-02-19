package speedtest

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestDownloadTestContext(t *testing.T) {
	GlobalDataManager.Reset()
	GlobalDataManager.SetRateCaptureFrequency(time.Millisecond * 100)
	GlobalDataManager.SetCaptureTime(time.Second)
	idealSpeed := 0.1 * 8 * float64(runtime.NumCPU()) * 10 / 0.1 // one mockRequest per second with all CPU cores
	delta := 0.05
	latency, _ := time.ParseDuration("5ms")
	server := Server{
		URL:     "https://dummy.com/upload.php",
		Latency: latency,
		Context: defaultClient,
	}

	err := server.downloadTestContext(
		context.Background(),
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
	GlobalDataManager.SetRateCaptureFrequency(time.Millisecond * 100)
	GlobalDataManager.SetCaptureTime(time.Second)

	idealSpeed := 0.1 * 8 * float64(runtime.NumCPU()) * 10 / 0.1 // one mockRequest per second with all CPU cores
	delta := 0.05                                                // tolerance scope (-0.05, +0.05)

	latency, _ := time.ParseDuration("5ms")
	server := Server{
		URL:     "https://dummy.com/upload.php",
		Latency: latency,
		Context: defaultClient,
	}

	err := server.uploadTestContext(
		context.Background(),
		mockRequest,
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	if server.ULSpeed < idealSpeed*(1-delta) || idealSpeed*(1+delta) < server.ULSpeed {
		t.Errorf("got unexpected server.ULSpeed '%v', expected between %v and %v", server.ULSpeed, idealSpeed*(1-delta), idealSpeed*(1+delta))
	}
}

func mockRequest(ctx context.Context, s *Server, w int) error {
	fmt.Sprintln(w)
	GlobalDataManager.SetRateCaptureFrequency(time.Millisecond * 100)
	GlobalDataManager.SetCaptureTime(time.Second)

	dc := GlobalDataManager.NewChunk()

	// (0.1MegaByte * 8bit * 8CPU * 10loop) / 0.1s = 640Megabit
	for i := 0; i < 10; i++ {
		dc.GetParent().AddTotalDownload(1 * 1000 * 100)
		dc.GetParent().AddTotalUpload(1 * 1000 * 100)
		time.Sleep(time.Millisecond * 10)
	}
	return nil
}

func TestPautaFilter(t *testing.T) {
	//vector := []float64{6, 6, 6, 6, 6, 6, 6, 6, 6, 6}
	vector0 := []int64{26, 23, 32}
	vector1 := []int64{3, 4, 5, 6, 6, 6, 1, 7, 9, 5, 200}
	_, _, std, _, _ := standardDeviation(vector0)
	if std != 3 {
		t.Fail()
	}

	result := pautaFilter(vector1)
	if len(result) != 10 {
		t.Fail()
	}
}
