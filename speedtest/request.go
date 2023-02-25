package speedtest

import (
	"context"
	"errors"
	"math"
	"net/http"
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

func (s *Server) MultiDownloadTestContext(ctx context.Context, servers Servers, savingMode bool) error {
	ss := servers.Available()
	if ss.Len() == 0 {
		return errors.New("not found available servers")
	}
	mainIDIndex := 0
	var fp *FuncGroup
	for i, server := range *ss {
		if server.ID == s.ID {
			mainIDIndex = i
		}
		sp := server
		fp = server.Context.RegisterDownloadHandler(func() {
			_ = downloadRequest(ctx, sp, 3)
		})
	}
	fp.Start(mainIDIndex) // block here
	//var serverPointer *Server = (*ss)[0]
	s.DLSpeed = fp.manager.GetAvgDownloadRate()
	return nil
}

func (s *Server) MultiUploadTestContext(ctx context.Context, servers Servers, savingMode bool) error {
	ss := servers.Available()
	if ss.Len() == 0 {
		return errors.New("not found available servers")
	}
	mainIDIndex := 0
	var fp *FuncGroup
	for i, server := range *ss {
		if server.ID == s.ID {
			mainIDIndex = i
		}
		sp := server
		fp = server.Context.RegisterUploadHandler(func() {
			_ = uploadRequest(ctx, sp, 3)
		})
	}
	fp.Start(mainIDIndex) // block here
	//var serverPointer *Server = (*ss)[0]
	s.ULSpeed = fp.manager.GetAvgUploadRate()
	return nil
}

// DownloadTest executes the test to measure download speed
func (s *Server) DownloadTest(savingMode bool) error {
	return s.downloadTestContext(context.Background(), downloadRequest)
}

// DownloadTestContext executes the test to measure download speed, observing the given context.
func (s *Server) DownloadTestContext(ctx context.Context) error {
	return s.downloadTestContext(ctx, downloadRequest)
}

func (s *Server) downloadTestContext(ctx context.Context, downloadRequest downloadFunc) error {
	s.Context.RegisterDownloadHandler(func() {
		_ = downloadRequest(ctx, s, 3)
	}).Start(0)
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
	s.Context.RegisterUploadHandler(func() {
		_ = uploadRequest(ctx, s, 4)
	}).Start(0)
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
	req.ContentLength = dc.(*DataChunk).ContentLength
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

	vectorPingResult, err := s.HTTPPing(ctx, pingURL, 10, time.Millisecond*200, nil)
	if err != nil || len(vectorPingResult) == 0 {
		return err
	}

	mean, _, std, min, max := standardDeviation(vectorPingResult)
	s.Latency = time.Duration(mean) * time.Nanosecond
	s.Jitter = time.Duration(std) * time.Nanosecond
	s.MinLatency = time.Duration(min) * time.Nanosecond
	s.MaxLatency = time.Duration(max) * time.Nanosecond
	return nil
}

func (s *Server) HTTPPing(
	ctx context.Context,
	dst string,
	echoTimes int,
	echoFreq time.Duration,
	callback func(latency time.Duration),
) ([]int64, error) {

	failTimes := 0
	var latencies []int64

	for i := 0; i < echoTimes; i++ {
		sTime := time.Now()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, dst, nil)
		if err != nil {
			return nil, err
		}

		resp, err := s.Context.doer.Do(req)

		endTime := time.Since(sTime)
		if err != nil {
			failTimes++
			continue
		}
		resp.Body.Close()
		latencies = append(latencies, endTime.Nanoseconds()/2)
		if callback != nil {
			callback(endTime)
		}
		time.Sleep(echoFreq)
	}
	if failTimes == echoTimes {
		return nil, errors.New("server connect timeout")
	}
	return latencies, nil
}

func (s *Server) ICMPPing(
	ctx context.Context,
	dst string,
	readTimeout int,
	echoOptionDataSize int,
	echoTimes int,
	echoFreq time.Duration,
	callback func(latency time.Duration),
) (latencies []int64, err error) {
	dialContext, err := s.Context.ipDialer.DialContext(ctx, "ip:icmp", dst)
	if err != nil {
		return nil, err
	}
	defer dialContext.Close()

	ICMPData := make([]byte, 8+echoOptionDataSize) // header + data
	ICMPData[0] = 8                                // echo
	ICMPData[1] = 0                                // code
	ICMPData[2] = 0                                // checksum
	ICMPData[3] = 0                                // checksum
	ICMPData[4] = 0                                // id
	ICMPData[5] = 1                                // id
	ICMPData[6] = 0                                // seq
	ICMPData[7] = 1                                // seq

	var echoMessage = "Hi! SpeedTest-Go \\(●'◡'●)/"

	for i := 0; i < len(echoMessage); i++ {
		ICMPData[7+i] = echoMessage[i]
	}

	failTimes := 0
	for i := 0; i < echoTimes; i++ {
		ICMPData[2] = byte(0)
		ICMPData[3] = byte(0)

		ICMPData[6] = byte(1 >> 8)
		ICMPData[7] = byte(1)
		ICMPData[8+echoOptionDataSize-1] = 6
		cs := checkSum(ICMPData)
		ICMPData[2] = byte(cs >> 8)
		ICMPData[3] = byte(cs)

		sTime := time.Now()
		_ = dialContext.SetDeadline(sTime.Add(time.Duration(readTimeout) * time.Millisecond))
		_, err = dialContext.Write(ICMPData)
		if err != nil {
			failTimes += echoTimes - i
			break
		}
		buf := make([]byte, 20+32+8)
		_, err = dialContext.Read(buf)
		if err != nil {
			failTimes++
			continue
		}
		endTime := time.Since(sTime)
		latencies = append(latencies, endTime.Nanoseconds())
		if callback != nil {
			callback(endTime)
		}
		time.Sleep(echoFreq)
	}
	if failTimes == echoTimes {
		return nil, errors.New("server connect timeout")
	}
	return
}

func checkSum(data []byte) uint16 {
	var sum uint32
	var length = len(data)
	var index int
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		index += 2
		length -= 2
	}
	if length > 0 {
		sum += uint32(data[index])
	}
	sum += sum >> 16
	return uint16(^sum)
}

func standardDeviation(vector []int64) (mean, variance, stdDev, min, max int64) {
	var sumNum, accumulate int64
	min = math.MaxInt64
	max = math.MinInt64
	for _, value := range vector {
		sumNum += value
		if min > value {
			min = value
		}
		if max < value {
			max = value
		}
	}
	mean = sumNum / int64(len(vector))
	for _, value := range vector {
		accumulate += (value - mean) * (value - mean)
	}
	variance = accumulate / int64(len(vector))
	stdDev = int64(math.Sqrt(float64(variance)))
	return
}
