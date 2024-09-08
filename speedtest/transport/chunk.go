package transport

import (
	"github.com/showwin/speedtest-go/speedtest/control"
	"io"
	"time"
)

// DataChunk The Speedtest's I/O implementation of Transmission Control Protocol.
type DataChunk struct {
	dateType            control.Proto
	startTime           time.Time
	endTime             time.Time
	err                 error
	ContentLength       int64
	remainOrDiscardSize int64
	ctrl                control.Controller
}

func NewChunk(controller control.Controller) control.Chunk {
	return &DataChunk{ctrl: controller}
}

// UploadHandler Create an upload handler
// @Related TCP UPLOAD
func (dc *DataChunk) UploadHandler(size int64) control.Chunk {
	if dc.dateType != control.TypeChunkUndefined {
		dc.err = control.ErrDuplicateCall
	}
	dc.ContentLength = size
	dc.remainOrDiscardSize = size
	dc.dateType = control.TypeUpload | control.TypeTCP
	dc.startTime = time.Now()
	return dc
}

func (dc *DataChunk) DownloadHandler(r io.Reader) error {
	if dc.dateType != control.TypeChunkUndefined {
		dc.err = control.ErrDuplicateCall
		return dc.err
	}
	dc.dateType = control.TypeDownload | control.TypeTCP
	dc.startTime = time.Now()
	defer func() {
		dc.endTime = time.Now()
	}()
	bufP := control.BlackHole.Get().(*[]byte)
	defer control.BlackHole.Put(bufP)
	readSize := 0
	for {
		select {
		case <-dc.ctrl.Done():
			return nil
		default:
			readSize, dc.err = r.Read(*bufP)
			rs := int64(readSize)

			dc.remainOrDiscardSize += rs
			dc.ctrl.Add(rs)
			if dc.err != nil {
				if dc.err == io.EOF {
					return nil
				}
				return dc.err
			}
		}
	}
}

func (dc *DataChunk) Rate() float64 {
	if dc.dateType.Assert(control.TypeDownload) {
		return float64(dc.remainOrDiscardSize) / dc.Duration().Seconds()
	} else if dc.dateType.Assert(control.TypeUpload) {
		return float64(dc.ContentLength-dc.remainOrDiscardSize) * 8 / 1000 / 1000 / dc.Duration().Seconds()
	}
	return 0
}

func (dc *DataChunk) Duration() time.Duration {
	return dc.endTime.Sub(dc.startTime)
}

func (dc *DataChunk) Type() control.Proto {
	return dc.dateType
}

func (dc *DataChunk) Len() int64 {
	return dc.ContentLength
}

// WriteTo Used to hook body traffic.
// @Related TCP UPLOAD
func (dc *DataChunk) WriteTo(w io.Writer) (written int64, err error) {
	nw := 0
	nr := control.DefaultReadChunkSize
	for {
		select {
		case <-dc.ctrl.Done():
			dc.endTime = time.Now()
			return written, io.EOF
		default:
			if dc.remainOrDiscardSize <= 0 {
				dc.endTime = time.Now()
				return written, io.EOF
			}
			if dc.remainOrDiscardSize < control.DefaultReadChunkSize {
				nr = int(dc.remainOrDiscardSize)
				nw, err = w.Write(dc.ctrl.Repeat()[:nr])
			} else {
				nw, err = w.Write(dc.ctrl.Repeat())
			}
			if err != nil {
				return
			}
			n64 := int64(nw)
			written += n64
			dc.remainOrDiscardSize -= n64
			dc.ctrl.Add(n64)
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
	}
}

// @Related TCP UPLOAD
func (dc *DataChunk) Read(b []byte) (n int, err error) {
	//TODO implement me
	panic("implement me")
}
