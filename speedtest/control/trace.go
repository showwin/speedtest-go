package control

const DefaultMaxTraceSize = 10

type Trace []Chunk

type Tracer struct {
	ts      []*Trace
	maxSize int
}

func NewHistoryTracer(size int) *Tracer {
	return &Tracer{
		ts:      make([]*Trace, 0, size),
		maxSize: size,
	}
}

func (rs *Tracer) Push(value Trace) {
	if len(rs.ts) == rs.maxSize {
		rs.ts = rs.ts[1:]
	}
	rs.ts = append(rs.ts, &value)
}

func (rs *Tracer) Latest() Trace {
	if len(rs.ts) > 0 {
		return *rs.ts[len(rs.ts)-1]
	}
	return nil
}

func (rs *Tracer) All() []*Trace {
	return rs.ts
}

func (rs *Tracer) Clean() {
	rs.ts = make([]*Trace, 0, rs.maxSize)
}
