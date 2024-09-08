package transport

const MaxBufferSize = 32 * 1000 * 1000 // 32 KB

type bufferScalar struct {
	factor      int64
	connections int64
}

func (s *bufferScalar) update(rate int64) int64 {
	ret := rate / s.factor / s.connections
	if ret > MaxBufferSize {
		return MaxBufferSize
	}
	return ret
}

func newBufAllocator(factor, connections int64) *bufferScalar {
	return &bufferScalar{factor: factor, connections: connections}
}
