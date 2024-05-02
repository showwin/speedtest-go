package speedtest

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestDownloadTestContext(t *testing.T) {
	idealSpeed := 0.1 * 8 * float64(runtime.NumCPU()) * 10 / 0.1 // one mockRequest per second with all CPU cores
	delta := 0.15
	latency, _ := time.ParseDuration("5ms")
	server := Server{
		URL:     "https://dummy.com/upload.php",
		Latency: latency,
		Context: defaultClient,
	}

	server.Context.Manager.Reset()
	server.Context.SetRateCaptureFrequency(time.Millisecond)
	server.Context.SetCaptureTime(time.Second)

	err := server.downloadTestContext(
		context.Background(),
		mockRequest,
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	value := server.Context.Manager.GetAvgDownloadRate()
	if value < idealSpeed*(1-delta) || idealSpeed*(1+delta) < value {
		t.Errorf("got unexpected server.DLSpeed '%v', expected between %v and %v", value, idealSpeed*(1-delta), idealSpeed*(1+delta))
	}
	if server.TestDuration.Download == nil || *server.TestDuration.Download != *server.TestDuration.Total {
		t.Errorf("can't count test duration, server.TestDuration.Download=%v, server.TestDuration.Total=%v", server.TestDuration.Download, server.TestDuration.Total)
	}
}

func TestUploadTestContext(t *testing.T) {
	idealSpeed := 0.1 * 8 * float64(runtime.NumCPU()) * 10 / 0.1 // one mockRequest per second with all CPU cores
	delta := 0.15                                                // tolerance scope (-0.05, +0.05)

	latency, _ := time.ParseDuration("5ms")
	server := Server{
		URL:     "https://dummy.com/upload.php",
		Latency: latency,
		Context: defaultClient,
	}

	server.Context.Manager.Reset()
	server.Context.SetRateCaptureFrequency(time.Millisecond)
	server.Context.SetCaptureTime(time.Second)

	err := server.uploadTestContext(
		context.Background(),
		mockRequest,
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	value := server.Context.Manager.GetAvgUploadRate()
	if value < idealSpeed*(1-delta) || idealSpeed*(1+delta) < value {
		t.Errorf("got unexpected server.ULSpeed '%v', expected between %v and %v", value, idealSpeed*(1-delta), idealSpeed*(1+delta))
	}
	if server.TestDuration.Upload == nil || *server.TestDuration.Upload != *server.TestDuration.Total {
		t.Errorf("can't count test duration, server.TestDuration.Upload=%v, server.TestDuration.Total=%v", server.TestDuration.Upload, server.TestDuration.Total)
	}
}

func mockRequest(ctx context.Context, s *Server, w int) error {
	fmt.Sprintln(w)
	dc := s.Context.Manager.NewChunk()
	// (0.1MegaByte * 8bit * nConn * 10loop) / 0.1s = n*80Megabit
	// sleep has bad deviation on windows
	// ref https://github.com/golang/go/issues/44343
	dc.GetParent().AddTotalDownload(1 * 1000 * 1000)
	dc.GetParent().AddTotalUpload(1 * 1000 * 1000)
	time.Sleep(time.Millisecond * 100)
	return nil
}

func TestPautaFilter(t *testing.T) {
	//vector := []float64{6, 6, 6, 6, 6, 6, 6, 6, 6, 6}
	vector0 := []int64{26, 23, 32}
	vector1 := []int64{3, 4, 5, 6, 6, 6, 1, 7, 9, 5, 200}
	_, _, std, _, _ := StandardDeviation(vector0)
	if std != 3 {
		t.Fail()
	}

	result := pautaFilter(vector1)
	if len(result) != 10 {
		t.Fail()
	}
}
