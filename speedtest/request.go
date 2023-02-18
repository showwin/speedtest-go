package speedtest

import (
	"context"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type (
	downloadFunc func(context.Context, *Server, int) error
	uploadFunc   func(context.Context, *Server, int) error
)

var (
	dlSizes = [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
	ulSizes = [...]int{100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, 4000} // kB
)

const testTime = time.Second * 10

// DownloadTest executes the test to measure download speed
func (s *Server) DownloadTest(savingMode bool) error {
	return s.downloadTestContext(context.Background(), downloadRequest)
}

// DownloadTestContext executes the test to measure download speed, observing the given context.
func (s *Server) DownloadTestContext(ctx context.Context) error {
	return s.downloadTestContext(ctx, downloadRequest)
}

func (s *Server) downloadTestContext(ctx context.Context, downloadRequest downloadFunc) error {
	s.Context.DownloadRateCaptureHandler(func() {
		_ = downloadRequest(ctx, s, 5)
	})
	s.DLSpeed = s.Context.GetAvgDownloadRate()
	return nil
}

// UploadTest executes the test to measure upload speed
func (s *Server) UploadTest(savingMode bool) error {
	return s.uploadTestContext(context.Background(), uploadRequest)
}

// UploadTestContext executes the test to measure upload speed, observing the given context.
func (s *Server) UploadTestContext(ctx context.Context, savingMode bool) error {
	return s.uploadTestContext(ctx, uploadRequest)
}

func (s *Server) uploadTestContext(ctx context.Context, uploadRequest uploadFunc) error {
	s.Context.UploadRateCaptureHandler(func() {
		_ = uploadRequest(ctx, s, 5)
	})
	s.ULSpeed = s.Context.GetAvgUploadRate()
	return nil
}

func downloadRequest(ctx context.Context, s *Server, w int) error {
	size := dlSizes[w]
	xdlURL := strings.Split(s.URL, "/upload.php")[0] + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xdlURL, nil)
	if err != nil {
		return err
	}

	resp, err := s.Context.doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return s.Context.NewChunk().DownloadHandler(resp.Body)
}

func uploadRequest(ctx context.Context, s *Server, w int) error {
	size := ulSizes[w]
	dc := s.Context.NewChunk().UploadHandler(int64(size*100-51) * 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, dc)
	req.ContentLength = dc.ContentLength
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := s.Context.doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return err
}

// PingTest executes test to measure latency
func (s *Server) PingTest() error {
	return s.PingTestContext(context.Background())
}

// PingTestContext executes test to measure latency, observing the given context.
func (s *Server) PingTestContext(ctx context.Context) error {
	pingURL := strings.Split(s.URL, "/upload.php")[0] + "/latency.txt"

	l := time.Second * 10
	for i := 0; i < 3; i++ {
		sTime := time.Now()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingURL, nil)
		if err != nil {
			return err
		}

		resp, err := s.Context.doer.Do(req)
		if err != nil {
			return err
		}

		fTime := time.Now()
		if fTime.Sub(sTime) < l {
			l = fTime.Sub(sTime)
		}

		resp.Body.Close()
	}

	s.Latency = time.Duration(l.Nanoseconds() / 2)

	return nil
}
