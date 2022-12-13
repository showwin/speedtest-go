package speedtest

import (
	"bytes"
	"io"
)

const readChunkSize = 1024 * 32 // 32 KBytes

type RepeatReader struct {
	ContentLength int64
	rs            []byte
	n             int
}

func NewRepeatReader(size int) *RepeatReader {
	if size <= 0 {
		panic("the size of repeated bytes should be > 0")
	}
	seqChunk := bytes.Repeat([]byte{0xAA}, readChunkSize) // uniformly distributed sequence of bits
	return &RepeatReader{rs: seqChunk, ContentLength: int64(size), n: size}
}

func (r *RepeatReader) Read(b []byte) (n int, err error) {
	if r.n < readChunkSize {
		if r.n <= 0 {
			return n, io.EOF
		}
		n = copy(b, r.rs[:r.n])
	} else {
		n = copy(b, r.rs)
	}
	r.n -= n
	return
}
