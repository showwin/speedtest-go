package speedtest

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

type downloadWarmUpFunc func(context.Context, *http.Client, string) error
type downloadFunc func(context.Context, *http.Client, string, int) error
type uploadWarmUpFunc func(context.Context, *http.Client, string) error
type uploadFunc func(context.Context, *http.Client, string, int) error

var dlSizes = [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
var ulSizes = [...]int{100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, 4000} //kB

// DownloadTest executes the test to measure download speed
func (s *Server) DownloadTest(savingMode bool) error {
	return s.downloadTestContext(context.Background(), savingMode, dlWarmUp, downloadRequest)
}

// DownloadTestContext executes the test to measure download speed, observing the given context.
func (s *Server) DownloadTestContext(ctx context.Context, savingMode bool) error {
	return s.downloadTestContext(ctx, savingMode, dlWarmUp, downloadRequest)
}

func (s *Server) downloadTestContext(
	ctx context.Context,
	savingMode bool,
	dlWarmUp downloadWarmUpFunc,
	downloadRequest downloadFunc,
) error {
	dlURL := strings.Split(s.URL, "/upload.php")[0]
	eg := errgroup.Group{}

	// Warming up
	sTime := time.Now()
	for i := 0; i < 2; i++ {
		eg.Go(func() error {
			return dlWarmUp(ctx, s.doer, dlURL)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	fTime := time.Now()

	// If the bandwidth is too large, the download sometimes finish earlier than the latency.
	// In this case, we ignore the the latency that is included server information.
	// This is not affected to the final result since this is a warm up test.
	timeToSpend := fTime.Sub(sTime.Add(s.Latency)).Seconds()
	if timeToSpend < 0 {
		timeToSpend = fTime.Sub(sTime).Seconds()
	}

	// 1.125MB for each request (750 * 750 * 2)
	wuSpeed := 1.125 * 8 * 2 / timeToSpend

	// Decide workload by warm up speed
	workload := 0
	weight := 0
	skip := false
	if savingMode {
		workload = 6
		weight = 3
	} else if 50.0 < wuSpeed {
		workload = 32
		weight = 6
	} else if 10.0 < wuSpeed {
		workload = 16
		weight = 4
	} else if 4.0 < wuSpeed {
		workload = 8
		weight = 4
	} else if 2.5 < wuSpeed {
		workload = 4
		weight = 4
	} else {
		skip = true
	}

	// Main speedtest
	dlSpeed := wuSpeed
	if !skip {
		sTime = time.Now()
		for i := 0; i < workload; i++ {
			eg.Go(func() error {
				return downloadRequest(ctx, s.doer, dlURL, weight)
			})
		}
		if err := eg.Wait(); err != nil {
			return err
		}
		fTime = time.Now()

		reqMB := dlSizes[weight] * dlSizes[weight] * 2 / 1000 / 1000
		dlSpeed = float64(reqMB) * 8 * float64(workload) / fTime.Sub(sTime).Seconds()
	}

	s.DLSpeed = dlSpeed
	return nil
}

// UploadTest executes the test to measure upload speed
func (s *Server) UploadTest(savingMode bool) error {
	return s.uploadTestContext(context.Background(), savingMode, ulWarmUp, uploadRequest)
}

// UploadTestContext executes the test to measure upload speed, observing the given context.
func (s *Server) UploadTestContext(ctx context.Context, savingMode bool) error {
	return s.uploadTestContext(ctx, savingMode, ulWarmUp, uploadRequest)
}
func (s *Server) uploadTestContext(
	ctx context.Context,
	savingMode bool,
	ulWarmUp uploadWarmUpFunc,
	uploadRequest uploadFunc,
) error {
	// Warm up
	sTime := time.Now()
	eg := errgroup.Group{}
	for i := 0; i < 2; i++ {
		eg.Go(func() error {
			return ulWarmUp(ctx, s.doer, s.URL)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	fTime := time.Now()
	// 1.0 MB for each request
	wuSpeed := 1.0 * 8 * 2 / fTime.Sub(sTime.Add(s.Latency)).Seconds()

	// Decide workload by warm up speed
	workload := 0
	weight := 0
	skip := false
	if savingMode {
		workload = 1
		weight = 7
	} else if 50.0 < wuSpeed {
		workload = 40
		weight = 9
	} else if 10.0 < wuSpeed {
		workload = 16
		weight = 9
	} else if 4.0 < wuSpeed {
		workload = 8
		weight = 9
	} else if 2.5 < wuSpeed {
		workload = 4
		weight = 5
	} else {
		skip = true
	}

	// Main speedtest
	ulSpeed := wuSpeed
	if !skip {
		sTime = time.Now()
		for i := 0; i < workload; i++ {
			eg.Go(func() error {
				return uploadRequest(ctx, s.doer, s.URL, weight)
			})
		}
		if err := eg.Wait(); err != nil {
			return err
		}
		fTime = time.Now()

		reqMB := float64(ulSizes[weight]) / 1000
		ulSpeed = reqMB * 8 * float64(workload) / fTime.Sub(sTime).Seconds()
	}

	s.ULSpeed = ulSpeed

	return nil
}

func dlWarmUp(ctx context.Context, doer *http.Client, dlURL string) error {
	size := dlSizes[2]
	xdlURL := dlURL + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xdlURL, nil)
	if err != nil {
		return err
	}

	resp, err := doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

func ulWarmUp(ctx context.Context, doer *http.Client, ulURL string) error {
	size := ulSizes[4]
	v := url.Values{}
	v.Add("content", strings.Repeat("0123456789", size*100-51))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ulURL, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

func downloadRequest(ctx context.Context, doer *http.Client, dlURL string, w int) error {
	size := dlSizes[w]
	xdlURL := dlURL + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xdlURL, nil)
	if err != nil {
		return err
	}

	resp, err := doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

func uploadRequest(ctx context.Context, doer *http.Client, ulURL string, w int) error {
	size := ulSizes[w]
	v := url.Values{}
	v.Add("content", strings.Repeat("0123456789", size*100-51))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ulURL, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(io.Discard, resp.Body)
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

		resp, err := s.doer.Do(req)
		if err != nil {
			return err
		}

		fTime := time.Now()
		if fTime.Sub(sTime) < l {
			l = fTime.Sub(sTime)
		}

		resp.Body.Close()
	}

	s.Latency = time.Duration(int64(l.Nanoseconds() / 2))

	return nil
}
