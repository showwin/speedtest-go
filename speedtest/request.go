package speedtest

import (
	"context"
	"errors"
	"fmt"
	"github.com/showwin/speedtest-go/speedtest/control"
	"github.com/showwin/speedtest-go/speedtest/internal"
	"github.com/showwin/speedtest-go/speedtest/transport"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type (
	testFunc func(context.Context, *TestDirection, *Server, int) error
)

var (
	dlSizes = [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
	ulSizes = [...]int{100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, 4000} // kB
)

var (
	ErrConnectTimeout = errors.New("server connect timeout")
)

func (s *Server) pullTest(
	ctx context.Context,
	directionType control.Proto,
	testFn testFunc,
	callback func(rate float64),
	servers Servers,
) (*TestDirection, error) {
	var availableServers *Servers
	if servers == nil {
		availableServers = &Servers{s}
	} else {
		availableServers = servers.Available()
	}

	if availableServers.Len() == 0 {
		return nil, errors.New("not found available servers")
	}
	direction := s.Context.NewDirection(ctx, directionType).SetSamplingCallback(callback)
	for _, server := range *availableServers {
		var priority int64 = 2
		if server.ID == s.ID {
			priority = 1
		}
		internal.DBG().Printf("[%d] Register Handler: %s\n", directionType, server.URL)
		sp := server
		direction.RegisterHandler(func() error {
			connectContext, cancel := context.WithTimeout(context.Background(), s.Context.estTimeout)
			defer cancel()
			//fmt.Println(sp.Host)
			return testFn(connectContext, direction, sp, 3)
		}, priority)
	}
	direction.Start()
	return direction, nil
}

func (s *Server) MultiDownloadTestContext(ctx context.Context, servers Servers, callback func(rate float64)) error {
	direction, err := s.pullTest(ctx, control.TypeDownload, downloadRequest, callback, servers)
	if err != nil {
		return err
	}
	s.DLSpeed = ByteRate(direction.EWMA())
	s.TestDuration.Download = &direction.Duration
	s.testDurationTotalCount()
	return nil
}

func (s *Server) MultiUploadTestContext(ctx context.Context, servers Servers, callback func(rate float64)) error {
	direction, err := s.pullTest(ctx, control.TypeUpload, uploadRequest, callback, servers)
	if err != nil {
		return err
	}
	s.ULSpeed = ByteRate(direction.EWMA())
	s.TestDuration.Download = &direction.Duration
	s.testDurationTotalCount()
	return nil
}

// DownloadTest executes the test to measure download speed
func (s *Server) DownloadTest(callback func(rate float64)) error {
	// usually, the connections handled by speedtest server only alive time < 1 minute.
	// we set it 30 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	return s.DownloadTestContext(ctx, callback)
}

// DownloadTestContext executes the test to measure download speed, observing the given context.
func (s *Server) DownloadTestContext(ctx context.Context, callback func(rate float64)) error {
	direction, err := s.pullTest(ctx, control.TypeDownload, downloadRequest, callback, nil)
	if err != nil {
		return err
	}
	s.DLSpeed = ByteRate(direction.EWMA())
	s.TestDuration.Download = &direction.Duration
	s.testDurationTotalCount()
	return nil
}

// UploadTest executes the test to measure upload speed
func (s *Server) UploadTest(callback func(rate float64)) error {
	return s.UploadTestContext(context.Background(), callback)
}

// UploadTestContext executes the test to measure upload speed, observing the given context.
func (s *Server) UploadTestContext(ctx context.Context, callback func(rate float64)) error {
	direction, err := s.pullTest(ctx, control.TypeUpload, uploadRequest, callback, nil)
	if err != nil {
		return err
	}
	s.ULSpeed = ByteRate(direction.EWMA())
	s.TestDuration.Upload = &direction.Duration
	s.testDurationTotalCount()
	return nil
}

func downloadRequest(ctx context.Context, direction *TestDirection, s *Server, w int) error {
	size := dlSizes[w]
	u, err := url.Parse(s.URL)
	if err != nil {
		return err
	}
	u.Path = path.Dir(u.Path)
	xdlURL := u.JoinPath(fmt.Sprintf("random%dx%d.jpg", size, size)).String()
	internal.DBG().Printf("XdlURL: %s\n", xdlURL)

	chunk := direction.NewChunk()

	if direction.proto.Assert(control.TypeTCP) {
		dialer := &net.Dialer{}
		client, err1 := transport.NewClient(dialer)
		if err1 != nil {
			return err1
		}
		err = client.Connect(context.TODO(), s.Host)
		if err != nil {
			return err
		}
		connReader, err1 := client.RegisterDownload(int64(size))
		if err1 != nil {
			return err1
		}
		return chunk.DownloadHandler(connReader)
	} else {
		// set est deadline
		// TODO: tmp usage, we must split speedtest config and speedtest result.
		estContext, cancel := context.WithTimeout(context.Background(), s.Context.estTimeout)
		defer cancel()
		req, err := http.NewRequestWithContext(estContext, http.MethodGet, xdlURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Connection", "Keep-Alive")
		resp, err := s.Context.doer.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return chunk.DownloadHandler(resp.Body)
	}
}

func uploadRequest(ctx context.Context, direction *TestDirection, s *Server, w int) error {
	size := ulSizes[w]
	chunk := direction.NewChunk()

	if direction.proto.Assert(control.TypeTCP) {
		var chunkScale int64 = 1
		chunkSize := 1000 * 1000 * chunkScale
		dialer := &net.Dialer{}
		client, err := transport.NewClient(dialer)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		//fmt.Println(s.Host)
		err = client.Connect(context.TODO(), "speedtestd.kpn.com:8080") // TODO: NEED fix
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		remainSize, err := client.RegisterUpload(chunkSize)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		dc := chunk.UploadHandler(remainSize)
		rc := io.NopCloser(dc)
		_, err = client.Upload(rc)

		return err
	} else {
		dc := chunk.UploadHandler(int64(size*100-51) * 10)
		// set est deadline
		// TODO: tmp usage, we must split speedtest config and speedtest result.
		estContext, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		req, err := http.NewRequestWithContext(estContext, http.MethodPost, s.URL, dc)
		if err != nil {
			return err
		}
		req.ContentLength = dc.Len()
		req.Header.Set("Content-Type", "application/octet-stream")
		internal.DBG().Printf("Len=%d, XulURL: %s\n", dc.Len(), s.URL)
		resp, err := s.Context.doer.Do(req)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		defer resp.Body.Close()
		return err
	}
}

// PingTest executes test to measure latency
func (s *Server) PingTest(callback func(latency time.Duration)) error {
	return s.PingTestContext(context.Background(), callback)
}

// PingTestContext executes test to measure latency, observing the given context.
func (s *Server) PingTestContext(ctx context.Context, callback func(latency time.Duration)) (err error) {
	start := time.Now()
	var vectorPingResult []int64
	if s.Context.config.PingMode.Assert(control.TypeTCP) {
		vectorPingResult, err = s.TCPPing(ctx, 10, time.Millisecond*200, callback)
	} else if s.Context.config.PingMode.Assert(control.TypeICMP) {
		vectorPingResult, err = s.ICMPPing(ctx, time.Second*4, 10, time.Millisecond*200, callback)
	} else {
		vectorPingResult, err = s.HTTPPing(ctx, 10, time.Millisecond*200, callback)
	}
	if err != nil || len(vectorPingResult) == 0 {
		return err
	}
	internal.DBG().Printf("Before StandardDeviation: %v\n", vectorPingResult)
	mean, _, std, minLatency, maxLatency := StandardDeviation(vectorPingResult)
	duration := time.Since(start)
	s.Latency = time.Duration(mean) * time.Nanosecond
	s.Jitter = time.Duration(std) * time.Nanosecond
	s.MinLatency = time.Duration(minLatency) * time.Nanosecond
	s.MaxLatency = time.Duration(maxLatency) * time.Nanosecond
	s.TestDuration.Ping = &duration
	s.testDurationTotalCount()
	return nil
}

// TestAll executes ping, download and upload tests one by one
func (s *Server) TestAll() error {
	err := s.PingTest(nil)
	if err != nil {
		return err
	}
	err = s.DownloadTest(nil)
	if err != nil {
		return err
	}
	return s.UploadTest(nil)
}

func (s *Server) TCPPing(
	ctx context.Context,
	echoTimes int,
	echoFreq time.Duration,
	callback func(latency time.Duration),
) (latencies []int64, err error) {
	var pingDst string
	if len(s.Host) == 0 {
		u, err := url.Parse(s.URL)
		if err != nil || len(u.Host) == 0 {
			return nil, err
		}
		pingDst = u.Host
	} else {
		pingDst = s.Host
	}
	failTimes := 0
	client, err := transport.NewClient(s.Context.tcpDialer)
	if err != nil {
		return nil, err
	}
	err = client.Connect(ctx, pingDst)
	if err != nil {
		return nil, err
	}
	for i := 0; i < echoTimes; i++ {
		latency, err := client.PingContext(ctx)
		if err != nil {
			failTimes++
			continue
		}
		latencies = append(latencies, latency)
		if callback != nil {
			callback(time.Duration(latency))
		}
		time.Sleep(echoFreq)
	}
	if failTimes == echoTimes {
		return nil, ErrConnectTimeout
	}
	return
}

func (s *Server) HTTPPing(
	ctx context.Context,
	echoTimes int,
	echoFreq time.Duration,
	callback func(latency time.Duration),
) (latencies []int64, err error) {
	var contextErr error
	u, err := url.Parse(s.URL)
	if err != nil || len(u.Host) == 0 {
		return nil, err
	}
	u.Path = path.Dir(u.Path)
	pingDst := u.JoinPath("latency.txt").String()
	internal.DBG().Printf("Echo: %s\n", pingDst)
	failTimes := 0
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingDst, nil)
	if err != nil {
		return nil, err
	}
	// carry out an extra request to warm up the connection and ensure the first request is not going to affect the
	// overall estimation
	echoTimes++
	for i := 0; i < echoTimes; i++ {
		sTime := time.Now()
		resp, err := s.Context.doer.Do(req)
		endTime := time.Since(sTime)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				contextErr = err
				break
			}

			failTimes++
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if i > 0 {
			latency := endTime.Nanoseconds()
			latencies = append(latencies, latency)
			internal.DBG().Printf("RTT: %d\n", latency)
			if callback != nil {
				callback(endTime)
			}
		}
		time.Sleep(echoFreq)
	}

	if contextErr != nil {
		return latencies, contextErr
	}

	if failTimes == echoTimes {
		return nil, ErrConnectTimeout
	}

	return
}

const PingTimeout = -1
const echoOptionDataSize = 32 // `echoMessage` need to change at same time

// ICMPPing privileged method
func (s *Server) ICMPPing(
	ctx context.Context,
	readTimeout time.Duration,
	echoTimes int,
	echoFreq time.Duration,
	callback func(latency time.Duration),
) (latencies []int64, err error) {
	u, err := url.ParseRequestURI(s.URL)
	if err != nil || len(u.Host) == 0 {
		return nil, err
	}
	internal.DBG().Printf("Echo: %s\n", strings.Split(u.Host, ":")[0])
	dialContext, err := s.Context.ipDialer.DialContext(ctx, "ip:icmp", strings.Split(u.Host, ":")[0])
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
		_ = dialContext.SetDeadline(sTime.Add(readTimeout))
		_, err = dialContext.Write(ICMPData)
		if err != nil {
			failTimes += echoTimes - i
			break
		}
		buf := make([]byte, 20+echoOptionDataSize+8)
		_, err = dialContext.Read(buf)
		if err != nil || buf[20] != 0x00 {
			failTimes++
			continue
		}
		endTime := time.Since(sTime)
		latencies = append(latencies, endTime.Nanoseconds())
		internal.DBG().Printf("1RTT: %s\n", endTime)
		if callback != nil {
			callback(endTime)
		}
		time.Sleep(echoFreq)
	}
	if failTimes == echoTimes {
		return nil, ErrConnectTimeout
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

func StandardDeviation(vector []int64) (mean, variance, stdDev, min, max int64) {
	if len(vector) == 0 {
		return
	}
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
