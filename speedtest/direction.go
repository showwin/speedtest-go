package speedtest

import (
	"bytes"
	"context"
	"github.com/showwin/speedtest-go/speedtest/control"
	"github.com/showwin/speedtest-go/speedtest/http"
	"github.com/showwin/speedtest-go/speedtest/transport"

	"github.com/showwin/speedtest-go/speedtest/internal"
	"sync"
	"sync/atomic"
	"time"
)

type TestDirection struct {
	ctx              context.Context
	testCancel       context.CancelFunc
	proto            control.Proto              // see [Proto]
	RateSequence     []int64                    // rate history sequence
	manager          control.Manager            // manager
	totalDataVolume  int64                      // total sent/received data volume
	welford          *internal.Welford          // std/EWMA/mean
	samplingCallback func(realTimeRate float64) // sampling callback
	trace            control.Trace              // detailed chunk data tracing
	loadBalancer     *control.LoadBalancer
	repeatBytes      []byte
	Duration         time.Duration
	sync.Mutex
}

func NewDataDirection(m control.Manager, proto control.Proto) *TestDirection {
	r := bytes.Repeat([]byte{0xAA}, control.DefaultReadChunkSize) // uniformly distributed sequence of bits
	return &TestDirection{
		proto:        proto,
		manager:      m,
		repeatBytes:  r,
		loadBalancer: control.NewLoadBalancer(),
		ctx:          context.TODO(),
	}
}

func (td *TestDirection) NewChunk() control.Chunk {
	var chunk control.Chunk
	if td.proto.Assert(control.TypeTCP) {
		chunk = transport.NewChunk(td)
	} else {
		chunk = http.NewChunk(td) // using HTTP as default protocol
	}
	td.Lock()
	defer td.Unlock()
	td.trace = append(td.trace, chunk)
	return chunk
}

// Trace returns tracing data
func (td *TestDirection) Trace() control.Trace {
	td.Lock()
	defer td.Unlock()
	return td.trace
}

// Avg Get the overall average speed in the test direction.
func (td *TestDirection) Avg() float64 {
	unit := float64(td.manager.GetSamplingDuration() / time.Millisecond)
	return float64(td.GetTotalDataVolume()*8/1000) / unit
}

// EWMA Get real-time EWMA and average weighted values.
func (td *TestDirection) EWMA() float64 {
	if td.welford != nil {
		return td.welford.EWMA()
	}
	internal.DBG().Println("warning: empty td.welford")
	return 0
}

// GetTotalDataVolume Read the data volume in the current direction.
func (td *TestDirection) GetTotalDataVolume() int64 {
	return atomic.LoadInt64(&td.totalDataVolume)
}

// AddTotalDataVolume Add the data volume in the current direction.
func (td *TestDirection) AddTotalDataVolume(delta int64) int64 {
	return atomic.AddInt64(&td.totalDataVolume, delta)
}

func (td *TestDirection) Add(delta int64) {
	td.AddTotalDataVolume(delta)
}

func (td *TestDirection) Get() int64 {
	return td.GetTotalDataVolume()
}

func (td *TestDirection) Done() <-chan struct{} {
	return td.ctx.Done()
}

func (td *TestDirection) Repeat() []byte {
	return td.repeatBytes
}

// SetSamplingCallback Sets an optional periodic sampling callback function
// for the TestDirection.
func (td *TestDirection) SetSamplingCallback(callback func(rate float64)) *TestDirection {
	td.samplingCallback = callback
	return td
}

// RegisterHandler Add a test function for TestDirection that sequences a
// size that depends on the maximum number of connections.
func (td *TestDirection) RegisterHandler(task control.Task, priority int64) *TestDirection {
	if td.loadBalancer.Len() < td.manager.GetMaxConnections() {
		td.loadBalancer.Add(task, priority)
	}
	return td
}

// Start The Load balancer for TestDirection.
func (td *TestDirection) Start() {
	if td.loadBalancer == nil {
		panic("loadBalancer is nil")
	}
	if td.loadBalancer.Len() == 0 {
		panic("empty task stack")
	}
	// sampling
	td.rateSampling()
	wg := sync.WaitGroup{}
	start := time.Now()
	for i := 0; i < td.manager.GetMaxConnections(); i++ {
		wg.Add(1)
		go func() {
			for {
				select {
				case <-td.Done():
					wg.Done()
					return
				default:
					td.loadBalancer.Dispatch()
				}
			}
		}()
	}
	wg.Wait()
	td.Duration = time.Since(start)
}

func (td *TestDirection) rateSampling() {
	ticker := time.NewTicker(td.manager.GetSamplingPeriod())
	var prevTotalDataVolume int64 = 0
	td.welford = internal.NewWelford(5*time.Second, td.manager.GetSamplingPeriod())
	sTime := time.Now()
	go func(t *time.Ticker) {
		defer t.Stop()
		for {
			select {
			case <-td.Done():
				internal.DBG().Println("RateSampler: ctx.Done from another goroutine")
				return
			case <-t.C:
				newTotalDataVolume := td.GetTotalDataVolume()
				deltaDataVolume := newTotalDataVolume - prevTotalDataVolume
				prevTotalDataVolume = newTotalDataVolume
				if deltaDataVolume != 0 {
					td.RateSequence = append(td.RateSequence, deltaDataVolume)
				}
				// anyway we update the measuring instrument
				globalAvg := (float64(td.GetTotalDataVolume())) / float64(time.Since(sTime).Milliseconds()) * 1000
				if td.welford.Update(globalAvg, float64(deltaDataVolume)) {
					// direction canceled early.
					td.testCancel()
					internal.DBG().Println("RateSampler: terminate due to early stop")
					return
				}
				// reports the current rate by callback.
				if td.samplingCallback != nil {
					td.samplingCallback(td.welford.EWMA())
				}
			}
		}
	}(ticker)
}
