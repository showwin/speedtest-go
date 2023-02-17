package speedtest

import (
	"bytes"
	"github.com/LyricTian/queue"
	"io"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const readChunkSize = 1024 * 32 // 32 KBytes

type DataType int32

const TypeDownload = 0
const TypeUpload = 1

type DataManager struct {
	totalDownload int64
	totalUpload   int64

	DownloadRateSequence []float64
	UploadRateSequence   []float64

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

func (dm *DataManager) DownloadRateCaptureHandler(fn func(v interface{})) {
	dm.testHandler(dm.downloadRateCapture, queue.NewJob("upLink", fn))
}

func (dm *DataManager) UploadRateCaptureHandler(fn func(v interface{})) {
	dm.testHandler(dm.uploadRateCapture, queue.NewJob("upLink", fn))
}

func (dm *DataManager) testHandler(captureFunc func() *time.Ticker, job queue.Jober) {
	// When the number of processor cores is equivalent to the processing program,
	// the processing efficiency reaches the highest level (VT is not considered).
	q := queue.NewQueue(10, dm.nThread)
	q.Run()

	ticker := captureFunc()
	time.AfterFunc(dm.captureTime, func() {
		ticker.Stop()
		q.Terminate()
	})

	for i := 0; i < 1000; i++ {
		q.Push(job)
	}
}

func (dm *DataManager) downloadRateCapture() *time.Ticker {
	return dm.rateCapture(dm.GetTotalDownload, &dm.DownloadRateSequence)
}

func (dm *DataManager) uploadRateCapture() *time.Ticker {
	return dm.rateCapture(dm.GetTotalUpload, &dm.UploadRateSequence)
}

func (dm *DataManager) rateCapture(rateFunc func() int64, dst *[]float64) *time.Ticker {
	ticker := time.NewTicker(dm.rateCaptureFrequency)
	oldTotal := rateFunc()
	step := float64(time.Second / dm.rateCaptureFrequency)
	go func() {
		for range ticker.C {
			newTotal := rateFunc()
			delta := newTotal - oldTotal
			oldTotal = newTotal
			rate := float64(delta) * 8 / 1000000 * step // 125000
			*dst = append(*dst, rate)
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
	dm.DownloadRateSequence = []float64{}
	dm.UploadRateSequence = []float64{}
	return dm.totalUpload
}

func (dm *DataManager) GetAvgDownloadRate() float64 {
	return calcMAFilter(dm.DownloadRateSequence)
}

func (dm *DataManager) GetAvgUploadRate() float64 {
	return calcMAFilter(dm.UploadRateSequence)
}

type DataChunk struct {
	manager   *DataManager
	dateType  DataType
	startTime time.Time
	endTime   time.Time
	dataSize  int64
	err       error

	ContentLength int64
	n             int
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

// DownloadSnapshotHandler No value will be returned here, because the error will interrupt the test.
// The error chunk is generally caused by the remote server actively closing the connection.
func (dc *DataChunk) DownloadSnapshotHandler(r io.Reader) error {
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
		dc.dataSize += rs
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

func (dc *DataChunk) UploadSnapshotHandler(size int) *DataChunk {
	if size <= 0 {
		panic("the size of repeated bytes should be > 0")
	}

	dc.ContentLength = int64(size)
	dc.n = size
	dc.dateType = TypeUpload

	if dc.manager.repeatByte == nil {
		r := bytes.Repeat([]byte{0xAA}, readChunkSize) // uniformly distributed sequence of bits
		dc.manager.repeatByte = &r
	}

	dc.startTime = time.Now()
	return dc
}

func (dc *DataChunk) Read(b []byte) (n int, err error) {
	if dc.n < readChunkSize {
		if dc.n <= 0 {
			dc.endTime = time.Now()
			return n, io.EOF
		}
		n = copy(b, (*dc.manager.repeatByte)[:dc.n])
	} else {
		n = copy(b, *dc.manager.repeatByte)
	}
	dc.n -= n
	atomic.AddInt64(&dc.manager.totalUpload, int64(n))
	return
}

// calcMAFilter Median-Averaging Filter
func calcMAFilter(list []float64) float64 {
	sum := 0.0
	n := len(list)
	if n == 0 {
		return 0
	}
	sort.Float64s(list)
	for i := 1; i < n-1; i++ {
		sum += list[i]
	}
	return sum / float64(n-2)
}

var GlobalDataManager = NewDataManager()
