package speedtest

import "net/http"

// Speedtest is a speedtest client.
type Speedtest struct {
	doer *http.Client
}

// Option is a function that can be passed to New to modify the Client.
type Option func(*Speedtest)

// WithDoer sets the http.Client used to make requests.
func WithDoer(doer *http.Client) Option {
	return func(s *Speedtest) {
		s.doer = doer
	}
}

// New creates a new speedtest client.
func New(opts ...Option) *Speedtest {
	s := &Speedtest{
		doer: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

var defaultClient = New()
