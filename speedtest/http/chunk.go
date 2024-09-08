package http

import (
	"github.com/showwin/speedtest-go/speedtest/control"
	"io"
	"time"
)

// DataChunk The Speedtest's I/O implementation of Hypertext Transfer Protocol.
type DataChunk struct {
	dateType            control.Proto
	startTime           time.Time
	endTime             time.Time
	err                 error
	ContentLength       int64
	remainOrDiscardSize int64
	control.Controller
}

func NewChunk(controller control.Controller) control.Chunk {
	return &DataChunk{Controller: controller}
}

func (dc *DataChunk) Len() int64 {
	return dc.ContentLength
}

// Duration Get chunk duration (start -> end)
func (dc *DataChunk) Duration() time.Duration {
	return dc.endTime.Sub(dc.startTime)
}

// Rate Get chunk avg rate
func (dc *DataChunk) Rate() float64 {
	if dc.dateType.Assert(control.TypeDownload) {
		return float64(dc.remainOrDiscardSize) / dc.Duration().Seconds()
	} else if dc.dateType.Assert(control.TypeUpload) {
		return float64(dc.ContentLength-dc.remainOrDiscardSize) * 8 / 1000 / 1000 / dc.Duration().Seconds()
	}
	return 0
}

func (dc *DataChunk) Type() control.Proto {
	return dc.dateType
}

// DownloadHandler No value will be returned here, because the error will interrupt the test.
// The error chunk is generally caused by the remote server actively closing the connection.
// @Related HTTP Download
func (dc *DataChunk) DownloadHandler(r io.Reader) error {
	if dc.dateType != control.TypeChunkUndefined {
		dc.err = control.ErrDuplicateCall
		return dc.err
	}
	dc.dateType = control.TypeDownload | control.TypeHTTP
	dc.startTime = time.Now()
	defer func() {
		dc.endTime = time.Now()
	}()
	bufP := control.BlackHole.Get().(*[]byte)
	defer control.BlackHole.Put(bufP)
	readSize := 0
	for {
		select {
		case <-dc.Done():
			return nil
		default:
			readSize, dc.err = r.Read(*bufP)
			rs := int64(readSize)

			dc.remainOrDiscardSize += rs
			dc.Add(rs)
			if dc.err != nil {
				if dc.err == io.EOF {
					return nil
				}
				return dc.err
			}
		}
	}
}

// UploadHandler Create an upload handler
// @Related HTTP UPLOAD
func (dc *DataChunk) UploadHandler(size int64) control.Chunk {
	if dc.dateType != control.TypeChunkUndefined {
		dc.err = control.ErrDuplicateCall
	}

	if size <= 0 {
		panic("the size of repeated bytes should be > 0")
	}

	dc.ContentLength = size
	dc.remainOrDiscardSize = size
	dc.dateType = control.TypeUpload | control.TypeHTTP
	dc.startTime = time.Now()
	return dc
}

func (dc *DataChunk) Read(b []byte) (n int, err error) {
	if dc.remainOrDiscardSize < control.DefaultReadChunkSize {
		if dc.remainOrDiscardSize <= 0 {
			dc.endTime = time.Now()
			return n, io.EOF
		}
		n = copy(b, dc.Repeat()[:dc.remainOrDiscardSize])
	} else {
		n = copy(b, dc.Repeat())
	}
	n64 := int64(n)
	dc.remainOrDiscardSize -= n64
	dc.Add(n64)
	return
}
