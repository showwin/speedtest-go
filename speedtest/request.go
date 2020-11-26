package speedtest

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var dlSizes = [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
var ulSizes = [...]int{100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, 4000} //kB
var client = http.Client{}

// DownloadTest executes the test to measure download speed
func (s *Server) DownloadTest() error {
	dlURL := strings.Split(s.URL, "/upload")[0]
	wg := new(sync.WaitGroup)

	// Warming up
	sTime := time.Now()
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go dlWarmUp(wg, dlURL)
	}
	wg.Wait()
	fTime := time.Now()
	// 1.125MB for each request (750 * 750 * 2)
	wuSpeed := 1.125 * 8 * 2 / fTime.Sub(sTime.Add(s.Latency)).Seconds()

	// Decide workload by warm up speed
	workload := 0
	weight := 0
	skip := false
	if 10.0 < wuSpeed {
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
	if skip == false {
		sTime = time.Now()
		for i := 0; i < workload; i++ {
			wg.Add(1)
			go downloadRequest(wg, dlURL, weight)
		}
		wg.Wait()
		fTime = time.Now()

		reqMB := dlSizes[weight] * dlSizes[weight] * 2 / 1000 / 1000
		dlSpeed = float64(reqMB) * 8 * float64(workload) / fTime.Sub(sTime).Seconds()
	}

	s.DLSpeed = dlSpeed
	return nil
}

// UploadTest executes the test to measure upload speed
func (s *Server) UploadTest() error {
	wg := new(sync.WaitGroup)

	// Warm up
	sTime := time.Now()
	wg = new(sync.WaitGroup)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go ulWarmUp(wg, s.URL)
	}
	wg.Wait()
	fTime := time.Now()
	// 1.0 MB for each request
	wuSpeed := 1.0 * 8 * 2 / fTime.Sub(sTime.Add(s.Latency)).Seconds()

	// Decide workload by warm up speed
	workload := 0
	weight := 0
	skip := false
	if 10.0 < wuSpeed {
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
	if skip == false {
		sTime = time.Now()
		for i := 0; i < workload; i++ {
			wg.Add(1)
			go uploadRequest(wg, s.URL)
		}
		wg.Wait()
		fTime = time.Now()

		reqMB := float64(ulSizes[weight]) / 1000
		ulSpeed = reqMB * 8 * float64(workload) / fTime.Sub(sTime).Seconds()
	}

	s.ULSpeed = ulSpeed

	return nil
}

func dlWarmUp(wg *sync.WaitGroup, dlURL string) {
	size := dlSizes[2]
	xdlURL := dlURL + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"

	resp, err := client.Get(xdlURL)
	checkError(err)
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	wg.Done()
}

func ulWarmUp(wg *sync.WaitGroup, ulURL string) {
	size := ulSizes[4]
	v := url.Values{}
	v.Add("content", strings.Repeat("0123456789", size*100-51))

	resp, err := client.PostForm(ulURL, v)
	checkError(err)
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	wg.Done()
}

func downloadRequest(wg *sync.WaitGroup, dlURL string, w int) {
	size := dlSizes[w]
	xdlURL := dlURL + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"

	resp, err := client.Get(xdlURL)
	checkError(err)
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	wg.Done()
}

func uploadRequest(wg *sync.WaitGroup, ulURL string) {
	size := ulSizes[9]
	v := url.Values{}
	v.Add("content", strings.Repeat("0123456789", size*100-51))

	resp, err := client.PostForm(ulURL, v)
	checkError(err)
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	wg.Done()
}

// PingTest executes test to measure latency
func (s *Server) PingTest() error {
	pingURL := strings.Split(s.URL, "/upload")[0] + "/latency.txt"

	l := time.Duration(100000000000) // 10sec
	for i := 0; i < 3; i++ {
		sTime := time.Now()
		resp, err := http.Get(pingURL)
		fTime := time.Now()
		if err != nil {
			return err
		}
		if fTime.Sub(sTime) < l {
			l = fTime.Sub(sTime)
		}
		resp.Body.Close()
	}

	s.Latency = time.Duration(int64(l.Nanoseconds()/2))

	return nil
}
