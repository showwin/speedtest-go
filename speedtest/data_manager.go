package speedtest

import (
	"bytes"
	"io"
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

	DataGroup []*DataChunk
	sync.Mutex

	repeatByte *[]byte
}

func NewDataManager() *DataManager {
	var ret DataManager
	return &ret
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

var GlobalDataManager = NewDataManager()
