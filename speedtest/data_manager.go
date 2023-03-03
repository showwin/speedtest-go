package speedtest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Manager interface {
	SetRateCaptureFrequency(duration time.Duration) Manager
	SetCaptureTime(duration time.Duration) Manager

	NewChunk() Chunk

	GetTotalDownload() int64
	GetTotalUpload() int64
	AddTotalDownload(value int64)
	AddTotalUpload(value int64)

	GetAvgDownloadRate() float64
	GetAvgUploadRate() float64

	CallbackDownloadRate(callback func(downRate float64)) *time.Ticker
	CallbackUploadRate(callback func(upRate float64)) *time.Ticker

	RegisterDownloadHandler(fn func()) *FuncGroup
	RegisterUploadHandler(fn func()) *FuncGroup

	// Wait for the upload or download task to end to avoid errors caused by core occupation
	Wait()
	Reset()

	SetNThread(n int) Manager
}

type Chunk interface {
	UploadHandler(size int64) Chunk
	DownloadHandler(r io.Reader) error

	GetRate() float64
	GetDuration() time.Duration
	GetParent() Manager

	Read(b []byte) (n int, err error)
}

const readChunkSize = 1024 * 32 // 32 KBytes

type DataType int32

const TypeEmptyChunk = 0
const TypeDownload = 1
const TypeUpload = 2

type FuncGroup struct {
	fns     []func()
	manager *DataManager
}

func (f *FuncGroup) Add(fn func()) {
	f.fns = append(f.fns, fn)
}

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

	running bool

	dFn *FuncGroup
	uFn *FuncGroup
}

func NewDataManager() *DataManager {
	ret := &DataManager{
		nThread:              runtime.NumCPU(),
		captureTime:          time.Second * 10,
		rateCaptureFrequency: time.Millisecond * 100,
	}
	ret.dFn = &FuncGroup{manager: ret}
	ret.uFn = &FuncGroup{manager: ret}
	return ret
}

func (dm *DataManager) CallbackDownloadRate(callback func(downRate float64)) *time.Ticker {
	ticker := time.NewTicker(dm.rateCaptureFrequency)
	oldDownTotal := dm.GetTotalDownload()
	unit := float64(time.Second / dm.rateCaptureFrequency)

	go func() {
		for range ticker.C {
			newDownTotal := dm.GetTotalDownload()
			delta := newDownTotal - oldDownTotal
			oldDownTotal = newDownTotal
			callback(float64(delta) * 8 / 1000000 * unit)
		}
	}()
	return ticker
}

func (dm *DataManager) CallbackUploadRate(callback func(upRate float64)) *time.Ticker {
	ticker := time.NewTicker(dm.rateCaptureFrequency)
	oldUpTotal := dm.GetTotalUpload()
	unit := float64(time.Second / dm.rateCaptureFrequency)

	go func() {
		for range ticker.C {
			newUpTotal := dm.GetTotalUpload()
			delta := newUpTotal - oldUpTotal
			oldUpTotal = newUpTotal
			callback(float64(delta) * 8 / 1000000 * unit)
		}
	}()
	return ticker
}

func (dm *DataManager) Wait() {
	oldDownTotal := dm.GetTotalDownload()
	oldUpTotal := dm.GetTotalUpload()
	for {
		time.Sleep(dm.rateCaptureFrequency)
		newDownTotal := dm.GetTotalDownload()
		newUpTotal := dm.GetTotalUpload()
		deltaDown := newDownTotal - oldDownTotal
		deltaUp := newUpTotal - oldUpTotal
		oldDownTotal = newDownTotal
		oldUpTotal = newUpTotal
		if deltaDown == 0 && deltaUp == 0 {
			return
		}
	}
}

func (dm *DataManager) RegisterUploadHandler(fn func()) *FuncGroup {
	if len(dm.uFn.fns) < dm.nThread {
		dm.uFn.Add(fn)
	}
	return dm.uFn
}

func (dm *DataManager) RegisterDownloadHandler(fn func()) *FuncGroup {
	if len(dm.dFn.fns) < dm.nThread {
		dm.dFn.Add(fn)
	}
	return dm.dFn
}

func (f *FuncGroup) Start(cancel context.CancelFunc, mainRequestHandlerIndex int) {
	if len(f.fns) == 0 {
		panic("empty task stack")
	}
	if mainRequestHandlerIndex > len(f.fns)-1 {
		mainRequestHandlerIndex = 0
	}
	mainLoadFactor := 0.1
	// When the number of processor cores is equivalent to the processing program,
	// the processing efficiency reaches the highest level (VT is not considered).
	mainN := int(mainLoadFactor * float64(len(f.fns)))
	if mainN == 0 {
		mainN = 1
	}
	if len(f.fns) == 1 {
		mainN = f.manager.nThread
	}
	auxN := f.manager.nThread - mainN
	dbg.Printf("Available fns: %d\n", len(f.fns))
	dbg.Printf("mainN: %d\n", mainN)
	dbg.Printf("auxN: %d\n", auxN)
	wg := sync.WaitGroup{}
	f.manager.running = true
	ticker := f.manager.rateCapture()
	time.AfterFunc(f.manager.captureTime, func() {
		ticker.Stop()
		f.manager.running = false
		cancel()
		dbg.Println("FuncGroup: Stop")
	})
	for i := 0; i < mainN; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if !f.manager.running {
					return
				}
				f.fns[mainRequestHandlerIndex]()
			}
		}()
	}
	for j := 0; j < auxN; {
		for i := range f.fns {
			if j == auxN {
				break
			}
			if i == mainRequestHandlerIndex {
				continue
			}
			wg.Add(1)
			t := i
			go func() {
				defer wg.Done()
				for {
					if !f.manager.running {
						return
					}
					f.fns[t]()
				}
			}()
			j++
		}
	}
	wg.Wait()
}

func (dm *DataManager) rateCapture() *time.Ticker {
	ticker := time.NewTicker(dm.rateCaptureFrequency)
	oldTotalDownload := dm.totalDownload
	oldTotalUpload := dm.totalUpload
	go func() {
		for range ticker.C {
			newTotalDownload := dm.totalDownload
			newTotalUpload := dm.totalUpload
			deltaDownload := newTotalDownload - oldTotalDownload
			deltaUpload := newTotalUpload - oldTotalUpload
			oldTotalDownload = newTotalDownload
			oldTotalUpload = newTotalUpload
			if deltaDownload != 0 {
				dm.DownloadRateSequence = append(dm.DownloadRateSequence, deltaDownload)
			}
			if deltaUpload != 0 {
				dm.UploadRateSequence = append(dm.UploadRateSequence, deltaUpload)
			}
		}
	}()
	return ticker
}

func (dm *DataManager) NewChunk() Chunk {
	var dc DataChunk
	dc.manager = dm
	dm.Lock()
	dm.DataGroup = append(dm.DataGroup, &dc)
	dm.Unlock()
	return &dc
}

func (dm *DataManager) AddTotalDownload(value int64) {
	atomic.AddInt64(&dm.totalDownload, value)
}

func (dm *DataManager) AddTotalUpload(value int64) {
	atomic.AddInt64(&dm.totalUpload, value)
}

func (dm *DataManager) GetTotalDownload() int64 {
	return dm.totalDownload
}

func (dm *DataManager) GetTotalUpload() int64 {
	return dm.totalUpload
}

func (dm *DataManager) SetRateCaptureFrequency(duration time.Duration) Manager {
	dm.rateCaptureFrequency = duration
	return dm
}

func (dm *DataManager) SetCaptureTime(duration time.Duration) Manager {
	dm.captureTime = duration
	return dm
}

func (dm *DataManager) SetNThread(n int) Manager {
	if n < 1 {
		dm.nThread = runtime.NumCPU()
	} else {
		dm.nThread = n
	}
	return dm
}

func (dm *DataManager) Reset() {
	dm.totalDownload = 0
	dm.totalUpload = 0
	dm.DataGroup = []*DataChunk{}
	dm.DownloadRateSequence = []int64{}
	dm.UploadRateSequence = []int64{}
	dm.dFn.fns = []func(){}
	dm.uFn.fns = []func(){}
}

func (dm *DataManager) GetAvgDownloadRate() float64 {
	unit := float64(dm.captureTime / time.Millisecond)
	return float64(dm.totalDownload*8/1000) / unit
}

func (dm *DataManager) GetAvgUploadRate() float64 {
	unit := float64(dm.captureTime / time.Millisecond)
	return float64(dm.totalUpload*8/1000) / unit
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

func (dc *DataChunk) GetRate() float64 {
	if dc.dateType == TypeDownload {
		return float64(dc.remainOrDiscardSize) / dc.GetDuration().Seconds()
	} else if dc.dateType == TypeUpload {
		return float64(dc.ContentLength-dc.remainOrDiscardSize) * 8 / 1000 / 1000 / dc.GetDuration().Seconds()
	}
	return 0
}

// DownloadHandler No value will be returned here, because the error will interrupt the test.
// The error chunk is generally caused by the remote server actively closing the connection.
func (dc *DataChunk) DownloadHandler(r io.Reader) error {
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
		if !dc.manager.running {
			return nil
		}
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

func (dc *DataChunk) UploadHandler(size int64) Chunk {
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

func (dc *DataChunk) GetParent() Manager {
	return dc.manager
}

func (dc *DataChunk) Read(b []byte) (n int, err error) {
	if !dc.manager.running {
		return n, io.EOF
	}
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
func _(list []int64) float64 {
	if len(list) == 0 {
		return 0
	}
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

func pautaFilter(vector []int64) []int64 {
	dbg.Println("Per capture unit")
	dbg.Printf("Raw Sequence len: %d\n", len(vector))
	dbg.Printf("Raw Sequence: %v\n", vector)
	if len(vector) == 0 {
		return vector
	}
	mean, _, std, _, _ := sampleVariance(vector)
	var retVec []int64
	for _, value := range vector {
		if math.Abs(float64(value-mean)) < float64(3*std) {
			retVec = append(retVec, value)
		}
	}
	dbg.Printf("Raw average: %dByte\n", mean)
	dbg.Printf("Pauta Sequence len: %d\n", len(retVec))
	dbg.Printf("Pauta Sequence: %v\n", retVec)
	return retVec
}

// sampleVariance sample Variance
func sampleVariance(vector []int64) (mean, variance, stdDev, min, max int64) {
	if len(vector) == 0 {
		return 0, 0, 0, 0, 0
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
	variance = accumulate / int64(len(vector)-1) // Bessel's correction
	stdDev = int64(math.Sqrt(float64(variance)))
	return
}

var GlobalDataManager = NewDataManager()
