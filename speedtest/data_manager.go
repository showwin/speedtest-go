package speedtest

import (
	"bytes"
	"errors"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const readChunkSize = 1024 * 32 // 32 KBytes

type DataType int32

const TypeEmptyChunk = 0
const TypeDownload = 1
const TypeUpload = 2

type DataManager struct {
	totalDownload int64
	totalUpload   int64

	DownloadRateSequence []int64
	UploadRateSequence   []int64

	DataGroup []*DataChunk
	sync.Mutex

	repeatByte *[]byte

	captureTime          time.Duration
	rateCaptureFrequency time.Duration
	nThread              int
}

func NewDataManager() *DataManager {
	ret := &DataManager{
		nThread:              runtime.NumCPU(),
		captureTime:          time.Second * 10,
		rateCaptureFrequency: time.Second,
	}
	return ret
}

func (dm *DataManager) Wait() {
	oldDownTotal := GlobalDataManager.GetTotalDownload()
	oldUpTotal := GlobalDataManager.GetTotalUpload()
	for {
		time.Sleep(dm.rateCaptureFrequency)
		newDownTotal := GlobalDataManager.GetTotalDownload()
		newUpTotal := GlobalDataManager.GetTotalUpload()
		deltaDown := newDownTotal - oldDownTotal
		deltaUp := newUpTotal - oldUpTotal
		oldDownTotal = newDownTotal
		oldUpTotal = newUpTotal
		if deltaDown == 0 && deltaUp == 0 {
			return
		}
	}
}

func (dm *DataManager) DownloadRateCaptureHandler(fn func()) {
	dm.testHandler(dm.downloadRateCapture, fn)
}

func (dm *DataManager) UploadRateCaptureHandler(fn func()) {
	dm.testHandler(dm.uploadRateCapture, fn)
}

func (dm *DataManager) testHandler(captureFunc func() *time.Ticker, fn func()) {
	ticker := captureFunc()
	running := true
	wg := sync.WaitGroup{}
	time.AfterFunc(dm.captureTime, func() {
		ticker.Stop()
		running = false
	})
	// When the number of processor cores is equivalent to the processing program,
	// the processing efficiency reaches the highest level (VT is not considered).
	for i := 0; i < dm.nThread; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if !running {
					return
				}
				fn()
			}
		}()
	}
	wg.Wait()
}

func (dm *DataManager) downloadRateCapture() *time.Ticker {
	return dm.rateCapture(dm.GetTotalDownload, &dm.DownloadRateSequence)
}

func (dm *DataManager) uploadRateCapture() *time.Ticker {
	return dm.rateCapture(dm.GetTotalUpload, &dm.UploadRateSequence)
}

func (dm *DataManager) rateCapture(rateFunc func() int64, dst *[]int64) *time.Ticker {
	ticker := time.NewTicker(dm.rateCaptureFrequency)
	oldTotal := rateFunc()
	go func() {
		for range ticker.C {
			newTotal := rateFunc()
			delta := newTotal - oldTotal
			oldTotal = newTotal
			*dst = append(*dst, delta)
		}
	}()
	return ticker
}

func (dm *DataManager) NewDataChunk() *DataChunk {
	var dc DataChunk
	dc.manager = dm
	dm.Lock()
	dm.DataGroup = append(dm.DataGroup, &dc)
	dm.Unlock()
	return &dc
}

func (dm *DataManager) GetTotalDownload() int64 {
	return dm.totalDownload
}

func (dm *DataManager) GetTotalUpload() int64 {
	return dm.totalUpload
}

func (dm *DataManager) SetRateCaptureFrequency(duration time.Duration) *DataManager {
	dm.rateCaptureFrequency = duration
	return dm
}

func (dm *DataManager) SetCaptureTime(duration time.Duration) *DataManager {
	dm.captureTime = duration
	return dm
}

func (dm *DataManager) SetNThread(n int) *DataManager {
	dm.nThread = n
	return dm
}

func (dm *DataManager) Reset() int64 {
	dm.totalDownload = 0
	dm.totalUpload = 0
	dm.DataGroup = []*DataChunk{}
	dm.DownloadRateSequence = []int64{}
	dm.UploadRateSequence = []int64{}
	return dm.totalUpload
}

func (dm *DataManager) GetAvgDownloadRate() float64 {
	unit := float64(time.Second / dm.rateCaptureFrequency)
	d := calcMAFilter(dm.DownloadRateSequence)
	return d * 8 / 1000000 * unit
}

func (dm *DataManager) GetAvgUploadRate() float64 {
	unit := float64(time.Second / dm.rateCaptureFrequency)
	d := calcMAFilter(dm.UploadRateSequence)
	return d * 8 / 1000000 * unit
}

type DataChunk struct {
	manager             *DataManager
	dateType            DataType
	startTime           time.Time
	endTime             time.Time
	err                 error
	ContentLength       int64
	remainOrDiscardSize int64
}

var blackHolePool = sync.Pool{
	New: func() any {
		b := make([]byte, 8192)
		return &b
	},
}

func (dc *DataChunk) GetDuration() time.Duration {
	return dc.endTime.Sub(dc.startTime)
}

func (dc *DataChunk) GetChunkRate() float64 {
	if dc.dateType == TypeDownload {
		return float64(dc.remainOrDiscardSize) / dc.GetDuration().Seconds()
	} else if dc.dateType == TypeUpload {
		return float64(dc.ContentLength-dc.remainOrDiscardSize) * 8 / 1000 / 1000 / dc.GetDuration().Seconds()
	}
	return 0
}

// DownloadSnapshotHandler No value will be returned here, because the error will interrupt the test.
// The error chunk is generally caused by the remote server actively closing the connection.
func (dc *DataChunk) DownloadSnapshotHandler(r io.Reader) error {
	if dc.dateType != TypeEmptyChunk {
		dc.err = errors.New("multiple calls to the same chunk handler are not allowed")
		return dc.err
	}
	dc.dateType = TypeDownload
	dc.startTime = time.Now()
	defer func() {
		dc.endTime = time.Now()
	}()
	bufP := blackHolePool.Get().(*[]byte)
	readSize := 0
	for {
		readSize, dc.err = r.Read(*bufP)
		rs := int64(readSize)
		dc.remainOrDiscardSize += rs
		atomic.AddInt64(&dc.manager.totalDownload, rs)
		if dc.err != nil {
			blackHolePool.Put(bufP)
			if dc.err == io.EOF {
				return nil
			}
			return dc.err
		}
	}
}

func (dc *DataChunk) UploadSnapshotHandler(size int64) *DataChunk {
	if dc.dateType != TypeEmptyChunk {
		dc.err = errors.New("multiple calls to the same chunk handler are not allowed")
	}
	if size <= 0 {
		panic("the size of repeated bytes should be > 0")
	}

	dc.ContentLength = size
	dc.remainOrDiscardSize = size
	dc.dateType = TypeUpload

	if dc.manager.repeatByte == nil {
		r := bytes.Repeat([]byte{0xAA}, readChunkSize) // uniformly distributed sequence of bits
		dc.manager.repeatByte = &r
	}

	dc.startTime = time.Now()
	return dc
}

func (dc *DataChunk) Read(b []byte) (n int, err error) {
	if dc.remainOrDiscardSize < readChunkSize {
		if dc.remainOrDiscardSize <= 0 {
			dc.endTime = time.Now()
			return n, io.EOF
		}
		n = copy(b, (*dc.manager.repeatByte)[:dc.remainOrDiscardSize])
	} else {
		n = copy(b, *dc.manager.repeatByte)
	}
	n64 := int64(n)
	dc.remainOrDiscardSize -= n64
	atomic.AddInt64(&dc.manager.totalUpload, n64)
	return
}

// calcMAFilter Median-Averaging Filter
func calcMAFilter(list []int64) float64 {
	var sum int64 = 0
	n := len(list)
	if n == 0 {
		return 0
	}

	length := len(list)
	for i := 0; i < length-1; i++ {
		for j := i + 1; j < length; j++ {
			if list[i] > list[j] {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	for i := 1; i < n-1; i++ {
		sum += list[i]
	}
	return float64(sum) / float64(n-2)
}

var GlobalDataManager = NewDataManager()
