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
	nThread              int
	uploadMaxWorkers     int

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
	activeWorkers   int32                       // current upload worker limit
	maxWorkers      int32                       // configured upload worker limit
	RateSequence    []int64                     // rate history sequence
	welford         *internal.Welford           // std/EWMA/mean
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
		nThread:              runtime.NumCPU(),
		captureTime:          time.Second * 15,
		rateCaptureFrequency: time.Millisecond * 50,
		Snapshot:             &Snapshot{},
		repeatByte:           &r,
		uploadMaxWorkers:     8,
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
	if len(dm.upload.fns) < dm.nThread {
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
	if td.TestType == typeUpload && td.manager.uploadMaxWorkers < workerLimit {
		workerLimit = td.manager.uploadMaxWorkers
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
	td.maxWorkers = int32(workerLimit)
	initialWorkers := td.maxWorkers
	if td.TestType == typeUpload {
		if td.manager.uploadMaxWorkers <= 8 {
			initialWorkers = int32(runtime.NumCPU())
			if initialWorkers > td.maxWorkers {
				initialWorkers = td.maxWorkers
			}
		}
	}
	atomic.StoreInt32(&td.activeWorkers, initialWorkers)
	if td.TestType == typeUpload {
		dbg.Printf("Upload workers: %d/%d\n", initialWorkers, td.maxWorkers)
	}

	// refresh once function
	once := sync.Once{}
	td.closeFunc = func() {
		once.Do(func() {
			stopCapture <- true
			close(stopCapture)
			td.manager.runningRW.Lock()
			td.manager.running = false
			td.manager.runningRW.Unlock()
			cancel()
			dbg.Println("FuncGroup: Stop")
		})
	}

	if td.TestType == typeUpload && td.maxWorkers > 1 {
		go td.adaptUploadWorkers()
	}
	time.AfterFunc(td.manager.captureTime, td.closeFunc)
	for i := 0; i < mainN; i++ {
		wg.Add(1)
		workerID := i
		go td.runWorker(&wg, workerID, td.fns[mainRequestHandlerIndex])
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
			workerID := mainN + j
			go td.runWorker(&wg, workerID, td.fns[t])
			j++
		}
	}
	wg.Wait()
}

func (td *TestDirection) runWorker(wg *sync.WaitGroup, workerID int, fn func()) {
	defer wg.Done()
	for {
		td.manager.runningRW.RLock()
		running := td.manager.running
		td.manager.runningRW.RUnlock()
		if !running {
			return
		}
		if !td.workerEnabled(workerID) {
			time.Sleep(time.Millisecond)
			continue
		}
		fn()
	}
}

func (td *TestDirection) workerEnabled(workerID int) bool {
	if td.TestType != typeUpload {
		return true
	}
	return int32(workerID) < atomic.LoadInt32(&td.activeWorkers)
}

func (td *TestDirection) adaptUploadWorkers() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	previousConfirmed := td.GetTotalDataVolume()
	previousAt := time.Now()
	bestRate := 0.0
	for range ticker.C {
		td.manager.runningRW.RLock()
		running := td.manager.running
		td.manager.runningRW.RUnlock()
		if !running {
			return
		}

		now := time.Now()
		confirmed := td.GetTotalDataVolume()
		delta := confirmed - previousConfirmed
		elapsed := now.Sub(previousAt).Seconds()
		previousConfirmed = confirmed
		previousAt = now
		if delta <= 0 || elapsed <= 0 {
			continue
		}

		rate := float64(delta) / elapsed
		active := atomic.LoadInt32(&td.activeWorkers)
		read := td.GetTotalReadVolume()
		backlog := read - confirmed
		confirmation := td.manager.GetUploadConfirmationRatio()
		if bestRate == 0 || rate > bestRate*1.05 {
			bestRate = rate
			if active < td.maxWorkers {
				active++
				atomic.StoreInt32(&td.activeWorkers, active)
				dbg.Printf("Upload workers: %d/%d (confirmed: %.2f MB/s, confirmation: %.1f%%, backlog: %.2f MB)\n", active, td.maxWorkers, rate/1000/1000, confirmation*100, float64(backlog)/1000/1000)
			}
			continue
		}

		if active > 1 && (rate < bestRate*0.85 || backlog > int64(active)*2*1024*1024) {
			active = reducedUploadWorkerCount(active, rate, bestRate, backlog)
			atomic.StoreInt32(&td.activeWorkers, active)
			dbg.Printf("Upload workers: %d/%d (confirmed: %.2f MB/s, confirmation: %.1f%%, backlog: %.2f MB)\n", active, td.maxWorkers, rate/1000/1000, confirmation*100, float64(backlog)/1000/1000)
			bestRate = rate
		}
	}
}

func reducedUploadWorkerCount(active int32, rate, bestRate float64, backlog int64) int32 {
	if active <= 1 {
		return 1
	}

	factor := 1.0
	if bestRate > 0 && rate < bestRate*0.85 {
		factor = math.Min(factor, rate/bestRate)
	}
	backlogLimit := int64(active) * 2 * 1024 * 1024
	if backlog > backlogLimit {
		factor = math.Min(factor, float64(backlogLimit)/float64(backlog))
	}
	if factor >= 1 {
		return active - 1
	}

	target := int32(math.Ceil(float64(active) * factor))
	if target >= active {
		target = active - 1
	}
	if target < 1 {
		return 1
	}
	return target
}

func (td *TestDirection) rateCapture() chan bool {
	ticker := time.NewTicker(td.manager.rateCaptureFrequency)
	var prevTotalDataVolume int64 = 0
	stopCapture := make(chan bool)
	td.welford = internal.NewWelford(5*time.Second, td.manager.rateCaptureFrequency)
	sTime := time.Now()
	go func(t *time.Ticker) {
		defer t.Stop()
		for {
			select {
			case <-t.C:
				newTotalDataVolume := td.GetTotalDataVolume()
				deltaDataVolume := newTotalDataVolume - prevTotalDataVolume
				prevTotalDataVolume = newTotalDataVolume
				if deltaDataVolume != 0 {
					td.RateSequence = append(td.RateSequence, deltaDataVolume)
				}
				// anyway we update the measuring instrument
				globalAvg := (float64(td.GetTotalDataVolume())) / float64(time.Since(sTime).Milliseconds()) * 1000
				rateSample := float64(deltaDataVolume)
				if td.TestType == typeUpload {
					// Upload bytes are confirmed per completed request, so use the
					// cumulative rate to avoid reporting a zero/spike sawtooth.
					rateSample = globalAvg * td.manager.rateCaptureFrequency.Seconds()
				}
				if td.welford.Update(globalAvg, rateSample) {
					go td.closeFunc()
				}
				// reports the current rate at the given rate
				if td.captureCallback != nil {
					td.captureCallback(ByteRate(td.welford.EWMA()))
				}
			case stop := <-stopCapture:
				if stop {
					return
				}
			}
		}
	}(ticker)
	return stopCapture
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

func (dm *DataManager) GetTotalUpload() int64 {
	return dm.upload.GetTotalDataVolume()
}

// GetUploadBacklog returns upload bytes read by the transport but not yet
// confirmed by a successful upload response.
func (dm *DataManager) GetUploadBacklog() int64 {
	backlog := dm.upload.GetTotalReadVolume() - dm.GetTotalUpload()
	if backlog < 0 {
		return 0
	}
	return backlog
}

// GetUploadConfirmationRatio returns server-confirmed upload bytes divided by
// bytes read from upload request bodies. It is an application-level estimate,
// not a TCP ACK counter.
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
		dm.nThread = runtime.NumCPU()
		dm.uploadMaxWorkers = 8
	} else {
		dm.nThread = n
		dm.uploadMaxWorkers = n
	}
	return dm
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

func (dm *DataManager) GetEWMADownloadRate() float64 {
	if dm.download.welford != nil {
		return dm.download.welford.EWMA()
	}
	return 0
}

func (dm *DataManager) GetAvgUploadRate() float64 {
	unit := float64(dm.captureTime / time.Millisecond)
	return float64(dm.upload.GetTotalDataVolume()*8/1000) / unit
}

func (dm *DataManager) GetEWMAUploadRate() float64 {
	if dm.upload.welford != nil {
		return dm.upload.welford.EWMA()
	}
	return 0
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
