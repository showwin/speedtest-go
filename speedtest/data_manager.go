package speedtest

import (
	"bytes"
	"context"
	"errors"
	"github.com/showwin/speedtest-go/speedtest/internal"
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
	GetUploadBacklog() int64
	GetUploadConfirmationRatio() float64
	AddTotalDownload(value int64)
	AddTotalUpload(value int64)

	GetAvgDownloadRate() float64
	GetAvgUploadRate() float64

	GetEWMADownloadRate() float64
	GetEWMAUploadRate() float64

	SetCallbackDownload(callback func(downRate ByteRate))
	SetCallbackUpload(callback func(upRate ByteRate))

	RegisterDownloadHandler(fn func()) *TestDirection
	RegisterUploadHandler(fn func()) *TestDirection

	// Wait for the upload or download task to end to avoid errors caused by core occupation
	Wait()
	Reset()
	Snapshots() *Snapshots

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

const readChunkSize = 1024 // 1 KBytes with higher frequency rate feedback

type DataType int32

const (
	typeEmptyChunk = iota
	typeDownload
	typeUpload
)

var (
	ErrorUninitializedManager = errors.New("uninitialized manager")
)

type funcGroup struct {
	fns []func()
}

func (f *funcGroup) Add(fn func()) {
	f.fns = append(f.fns, fn)
}

type DataManager struct {
	SnapshotStore *Snapshots
	Snapshot      *Snapshot
	sync.Mutex

	repeatByte *[]byte

	captureTime          time.Duration
	rateCaptureFrequency time.Duration
	nThread              int // download connections
	nUploadThread        int // upload connections

	running   bool
	runningRW sync.RWMutex

	download *TestDirection
	upload   *TestDirection
}

type TestDirection struct {
	TestType        int                         // test type
	manager         *DataManager                // manager
	totalDataVolume int64                       // total send/receive data volume
	totalReadVolume int64                       // upload bytes read into the transport
	acked           ackedBytes                  // upload bytes acknowledged by the server's TCP stack
	RateSequence    []int64                     // rate history sequence
	currentRate     atomic.Uint64               // float64 bits of the latest rate estimate, bytes per second
	finalRateValue  float64                     // measured bytes per second at test termination
	captureCallback func(realTimeRate ByteRate) // user callback
	closeFunc       func()                      // close func
	*funcGroup                                  // actually exec function
}

func (dm *DataManager) NewDataDirection(testType int) *TestDirection {
	return &TestDirection{
		TestType:  testType,
		manager:   dm,
		funcGroup: &funcGroup{},
	}
}

func NewDataManager() *DataManager {
	r := bytes.Repeat([]byte{0xAA}, readChunkSize) // uniformly distributed sequence of bits
	ret := &DataManager{
		nThread:              defaultDownloadConnections(),
		nUploadThread:        defaultUploadConnections,
		captureTime:          time.Second * 15,
		rateCaptureFrequency: time.Millisecond * 50,
		Snapshot:             &Snapshot{},
		repeatByte:           &r,
	}
	ret.download = ret.NewDataDirection(typeDownload)
	ret.upload = ret.NewDataDirection(typeUpload)
	ret.SnapshotStore = newHistorySnapshots(maxSnapshotSize)
	return ret
}

func (dm *DataManager) SetCallbackDownload(callback func(downRate ByteRate)) {
	if dm.download != nil {
		dm.download.captureCallback = callback
	}
}

func (dm *DataManager) SetCallbackUpload(callback func(upRate ByteRate)) {
	if dm.upload != nil {
		dm.upload.captureCallback = callback
	}
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

func (dm *DataManager) RegisterUploadHandler(fn func()) *TestDirection {
	if len(dm.upload.fns) < dm.nUploadThread {
		dm.upload.Add(fn)
	}
	return dm.upload
}

func (dm *DataManager) RegisterDownloadHandler(fn func()) *TestDirection {
	if len(dm.download.fns) < dm.nThread {
		dm.download.Add(fn)
	}
	return dm.download
}

func (td *TestDirection) GetTotalDataVolume() int64 {
	return atomic.LoadInt64(&td.totalDataVolume)
}

func (td *TestDirection) AddTotalDataVolume(delta int64) int64 {
	return atomic.AddInt64(&td.totalDataVolume, delta)
}

func (td *TestDirection) GetTotalReadVolume() int64 {
	return atomic.LoadInt64(&td.totalReadVolume)
}

func (td *TestDirection) AddTotalReadVolume(delta int64) int64 {
	return atomic.AddInt64(&td.totalReadVolume, delta)
}

func (td *TestDirection) Start(cancel context.CancelFunc, mainRequestHandlerIndex int) {
	if len(td.fns) == 0 {
		panic("empty task stack")
	}
	if mainRequestHandlerIndex > len(td.fns)-1 {
		mainRequestHandlerIndex = 0
	}
	mainLoadFactor := 0.1
	workerLimit := td.manager.nThread
	if td.TestType == typeUpload {
		workerLimit = td.manager.nUploadThread
	}
	// When the number of processor cores is equivalent to the processing program,
	// the processing efficiency reaches the highest level (VT is not considered).
	mainN := int(mainLoadFactor * float64(len(td.fns)))
	if mainN == 0 {
		mainN = 1
	}
	if len(td.fns) == 1 {
		mainN = workerLimit
	}
	auxN := workerLimit - mainN
	dbg.Printf("Available fns: %d\n", len(td.fns))
	dbg.Printf("mainN: %d\n", mainN)
	dbg.Printf("auxN: %d\n", auxN)
	wg := sync.WaitGroup{}
	td.manager.running = true
	stopCapture := td.rateCapture()
	// refresh once function
	once := sync.Once{}
	td.closeFunc = func() {
		once.Do(func() {
			stopCapture()
			td.manager.runningRW.Lock()
			td.manager.running = false
			td.manager.runningRW.Unlock()
			cancel()
			dbg.Println("FuncGroup: Stop")
		})
	}

	time.AfterFunc(td.manager.captureTime, td.closeFunc)
	for i := 0; i < mainN; i++ {
		wg.Add(1)
		go td.runWorker(&wg, td.fns[mainRequestHandlerIndex])
	}
	for j := 0; j < auxN; {
		for i := range td.fns {
			if j == auxN {
				break
			}
			if i == mainRequestHandlerIndex {
				continue
			}
			wg.Add(1)
			t := i
			go td.runWorker(&wg, td.fns[t])
			j++
		}
	}
	wg.Wait()
}

func (td *TestDirection) runWorker(wg *sync.WaitGroup, fn func()) {
	defer wg.Done()
	for {
		td.manager.runningRW.RLock()
		running := td.manager.running
		td.manager.runningRW.RUnlock()
		if !running {
			return
		}
		fn()
	}
}

const (
	minCaptureTime = 4 * time.Second // the earliest a settled rate may end the test
	settledWithin  = 0.01            // relative tolerance of a settled rate
)

// measuredBytes is the meter's input: bytes received for download, and for upload the bytes
// the server's TCP stack has acknowledged, or confirmed request bytes where the platform does
// not report acknowledgements.
func (td *TestDirection) measuredBytes() int64 {
	if td.TestType == typeUpload {
		if acked, ok := td.acked.total(); ok {
			return acked
		}
	}
	return td.GetTotalDataVolume()
}

func (td *TestDirection) rateCapture() func() {
	ticker := time.NewTicker(td.manager.rateCaptureFrequency)
	var prevTotalDataVolume int64 = 0
	stopCapture := make(chan struct{})
	captureStopped := make(chan struct{})
	meter := &internal.Meter{}
	sTime := time.Now()
	go func(t *time.Ticker) {
		defer t.Stop()
		defer close(captureStopped)
		for {
			select {
			case <-t.C:
				newTotalDataVolume := td.GetTotalDataVolume()
				deltaDataVolume := newTotalDataVolume - prevTotalDataVolume
				prevTotalDataVolume = newTotalDataVolume
				if deltaDataVolume != 0 {
					td.RateSequence = append(td.RateSequence, deltaDataVolume)
				}
				elapsed := time.Since(sTime)
				meter.Add(elapsed.Seconds(), td.measuredBytes())
				rate := meter.Rate()
				td.currentRate.Store(math.Float64bits(rate))
				if elapsed >= minCaptureTime && meter.Settled(settledWithin) {
					go td.closeFunc()
				}
				if td.captureCallback != nil {
					td.captureCallback(ByteRate(rate))
				}
			case <-stopCapture:
				meter.Add(time.Since(sTime).Seconds(), td.measuredBytes())
				td.finalRateValue = meter.Rate()
				return
			}
		}
	}(ticker)
	return func() {
		close(stopCapture)
		<-captureStopped
	}
}

func (td *TestDirection) finalRate() float64 {
	return td.finalRateValue
}

func (td *TestDirection) rate() float64 {
	return math.Float64frombits(td.currentRate.Load())
}

func (dm *DataManager) NewChunk() Chunk {
	var dc DataChunk
	dc.manager = dm
	dm.Lock()
	*dm.Snapshot = append(*dm.Snapshot, &dc)
	dm.Unlock()
	return &dc
}

func (dm *DataManager) AddTotalDownload(value int64) {
	dm.download.AddTotalDataVolume(value)
}

func (dm *DataManager) AddTotalUpload(value int64) {
	dm.upload.AddTotalDataVolume(value)
}

func (dm *DataManager) GetTotalDownload() int64 {
	return dm.download.GetTotalDataVolume()
}

// GetTotalUpload returns the upload bytes the server has received: the bytes its TCP stack
// has acknowledged where the platform reports them, and otherwise the bytes confirmed by
// successful upload responses.
func (dm *DataManager) GetTotalUpload() int64 {
	return dm.upload.measuredBytes()
}

// GetUploadBacklog returns upload bytes read by the transport that the server has not yet
// received.
func (dm *DataManager) GetUploadBacklog() int64 {
	backlog := dm.upload.GetTotalReadVolume() - dm.GetTotalUpload()
	if backlog < 0 {
		return 0
	}
	return backlog
}

// GetUploadConfirmationRatio returns the upload bytes the server has received divided by
// the bytes read from upload request bodies.
func (dm *DataManager) GetUploadConfirmationRatio() float64 {
	read := dm.upload.GetTotalReadVolume()
	if read <= 0 {
		return 0
	}
	confirmed := dm.GetTotalUpload()
	ratio := float64(confirmed) / float64(read)
	if ratio > 1 {
		return 1
	}
	if ratio < 0 {
		return 0
	}
	return ratio
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
		dm.nThread, dm.nUploadThread = defaultDownloadConnections(), defaultUploadConnections
	} else {
		dm.nThread, dm.nUploadThread = n, n
	}
	return dm
}

// The default connection counts keep the load public servers already see: download uses one
// connection per CPU and upload uses 8. Far servers cap each connection's rate, so long links
// fill only with more connections, which SetNThread allows.
const defaultUploadConnections = 8

func defaultDownloadConnections() int {
	return runtime.NumCPU()
}

func (dm *DataManager) Snapshots() *Snapshots {
	return dm.SnapshotStore
}

func (dm *DataManager) Reset() {
	dm.SnapshotStore.push(dm.Snapshot)
	dm.Snapshot = &Snapshot{}
	dm.download = dm.NewDataDirection(typeDownload)
	dm.upload = dm.NewDataDirection(typeUpload)
}

func (dm *DataManager) GetAvgDownloadRate() float64 {
	unit := float64(dm.captureTime / time.Millisecond)
	return float64(dm.download.GetTotalDataVolume()*8/1000) / unit
}

// GetEWMADownloadRate returns the current download rate in bytes per second, measured over the last
// two thirds of the test so far. The name predates the measurement, which is no longer an EWMA.
func (dm *DataManager) GetEWMADownloadRate() float64 {
	return dm.download.rate()
}

func (dm *DataManager) GetAvgUploadRate() float64 {
	unit := float64(dm.captureTime / time.Millisecond)
	return float64(dm.upload.GetTotalDataVolume()*8/1000) / unit
}

// GetEWMAUploadRate returns the current upload rate in bytes per second, measured over the last
// two thirds of the test so far. The name predates the measurement, which is no longer an EWMA.
func (dm *DataManager) GetEWMAUploadRate() float64 {
	return dm.upload.rate()
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
	switch dc.dateType {
	case typeDownload:
		return float64(dc.remainOrDiscardSize) / dc.GetDuration().Seconds()
	case typeUpload:
		return float64(dc.ContentLength-dc.remainOrDiscardSize) * 8 / 1000 / 1000 / dc.GetDuration().Seconds()
	default:
		return 0
	}
}

// DownloadHandler No value will be returned here, because the error will interrupt the test.
// The error chunk is generally caused by the remote server actively closing the connection.
func (dc *DataChunk) DownloadHandler(r io.Reader) error {
	if dc.dateType != typeEmptyChunk {
		dc.err = errors.New("multiple calls to the same chunk handler are not allowed")
		return dc.err
	}
	dc.dateType = typeDownload
	dc.startTime = time.Now()
	defer func() {
		dc.endTime = time.Now()
	}()
	bufP := blackHolePool.Get().(*[]byte)
	defer blackHolePool.Put(bufP)
	readSize := 0
	for {
		dc.manager.runningRW.RLock()
		running := dc.manager.running
		dc.manager.runningRW.RUnlock()
		if !running {
			return nil
		}
		readSize, dc.err = r.Read(*bufP)
		rs := int64(readSize)

		dc.remainOrDiscardSize += rs
		dc.manager.download.AddTotalDataVolume(rs)
		if dc.err != nil {
			if dc.err == io.EOF {
				return nil
			}
			return dc.err
		}
	}
}

func (dc *DataChunk) UploadHandler(size int64) Chunk {
	if dc.dateType != typeEmptyChunk {
		dc.err = errors.New("multiple calls to the same chunk handler are not allowed")
	}

	if size <= 0 {
		panic("the size of repeated bytes should be > 0")
	}

	dc.ContentLength = size
	dc.remainOrDiscardSize = size
	dc.dateType = typeUpload
	dc.startTime = time.Now()
	return dc
}

func (dc *DataChunk) GetParent() Manager {
	return dc.manager
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
	if dc.dateType == typeUpload {
		dc.manager.upload.AddTotalReadVolume(n64)
	}
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

const maxSnapshotSize = 10

type Snapshot []*DataChunk

type Snapshots struct {
	sp      []*Snapshot
	maxSize int
}

func newHistorySnapshots(size int) *Snapshots {
	return &Snapshots{
		sp:      make([]*Snapshot, 0, size),
		maxSize: size,
	}
}

func (rs *Snapshots) push(value *Snapshot) {
	if len(rs.sp) == rs.maxSize {
		rs.sp = rs.sp[1:]
	}
	rs.sp = append(rs.sp, value)
}

func (rs *Snapshots) Latest() *Snapshot {
	if len(rs.sp) > 0 {
		return rs.sp[len(rs.sp)-1]
	}
	return nil
}

func (rs *Snapshots) All() []*Snapshot {
	return rs.sp
}

func (rs *Snapshots) Clean() {
	rs.sp = make([]*Snapshot, 0, rs.maxSize)
}
