package speedtest

import (
	"context"
	"errors"
	"github.com/showwin/speedtest-go/speedtest/control"
	"runtime"
	"time"
)

var (
	ErrorUninitializedManager = errors.New("uninitialized manager")
)

type DataManager struct {
	// protocol indicates the transport that the manager should use.
	// Optional:
	// [control.TypeTCP]
	// [control.TypeHTTP]
	protocol control.Proto

	// estTimeout refers to the timeout threshold when establishing a connection.
	// By default, we consider the connection timed out when the handshake takes
	// more than 4 seconds.
	estTimeout time.Duration

	// samplingPeriod indicates the sampling period of the sampler.
	samplingPeriod time.Duration

	// samplingDuration indicates the maximum sampling duration of the sampler.
	samplingDuration time.Duration

	// maxConnections refers to the maximum number of connections, the default
	// is the number of logical cores of the device.
	// It is recommended to set maxConnections = 8.
	maxConnections int
	Tracer         *control.Tracer
}

func NewDataManager(protocol control.Proto) *DataManager {
	return &DataManager{
		maxConnections:   runtime.NumCPU(),
		estTimeout:       time.Second * 4,
		samplingPeriod:   time.Second * 15,
		samplingDuration: time.Millisecond * 50,
		Tracer:           control.NewHistoryTracer(control.DefaultMaxTraceSize),
		protocol:         protocol,
	}
}

// NewDirection
// @param ctx indicates the deadline of the sampler.
// timeout should not be greater than [DataManager.samplingDuration].
func (dm *DataManager) NewDirection(ctx context.Context, testDirection control.Proto) *TestDirection {
	direction := NewDataDirection(dm, dm.protocol|testDirection)
	dm.Tracer.Push(direction.Trace())
	direction.ctx, direction.testCancel = context.WithCancel(ctx)
	return direction
}

func (dm *DataManager) GetMaxConnections() int {
	return dm.maxConnections
}

func (dm *DataManager) SetSamplingPeriod(duration time.Duration) control.Manager {
	dm.samplingDuration = duration
	return dm
}

func (dm *DataManager) SetSamplingDuration(duration time.Duration) control.Manager {
	dm.samplingPeriod = duration
	return dm
}

func (dm *DataManager) GetSamplingPeriod() time.Duration {
	return dm.samplingDuration
}

func (dm *DataManager) GetSamplingDuration() time.Duration {
	return dm.samplingPeriod
}

func (dm *DataManager) SetNThread(n int) control.Manager {
	return dm.SetMaxConnections(n)
}

func (dm *DataManager) SetMaxConnections(n int) control.Manager {
	if n < 1 {
		dm.maxConnections = runtime.NumCPU()
	} else {
		dm.maxConnections = n
	}
	return dm
}

func (dm *DataManager) History() *control.Tracer {
	return dm.Tracer
}
