package control

import (
	"errors"
	"io"
	"sync"
	"time"
)

const DefaultReadChunkSize = 1024 // 1 KBytes with higher frequency rate feedback

var (
	ErrDuplicateCall = errors.New("multiple calls to the same chunk handler are not allowed")
)

type Chunk interface {
	UploadHandler(size int64) Chunk
	DownloadHandler(r io.Reader) error

	Rate() float64
	Duration() time.Duration

	Type() Proto

	Len() int64

	Read(b []byte) (n int, err error)
}

var BlackHole = sync.Pool{
	New: func() any {
		b := make([]byte, 8192)
		return &b
	},
}
