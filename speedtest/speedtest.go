package speedtest

import (
	"fmt"
	"net/http"
)

var (
	version          = "1.3.0"
	defaultUserAgent = fmt.Sprintf("showwin/speedtest-go %s", version)
)

// Speedtest is a speedtest client.
type Speedtest struct {
	doer *http.Client
}

type userAgentTransport struct {
	T         http.RoundTripper
	UserAgent string
}

func newUserAgentTransport(T http.RoundTripper, UserAgent string) *userAgentTransport {
	if T == nil {
		T = http.DefaultTransport
	}
	return &userAgentTransport{T, UserAgent}
}

func (uat *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", uat.UserAgent)
	return uat.T.RoundTrip(req)
}

// Option is a function that can be passed to New to modify the Client.
type Option func(*Speedtest)

// WithDoer sets the http.Client used to make requests.
func WithDoer(doer *http.Client) Option {
	return func(s *Speedtest) {
		s.doer = doer
	}
}

// WithUserAgent adds the passed "User-Agent" header to all requests.
// To use with a custom Doer, "WithDoer" must be passed before WithUserAgent:
// `New(WithDoer(myDoer), WithUserAgent(myUserAgent))`
func WithUserAgent(UserAgent string) Option {
	return func(s *Speedtest) {
		s.doer.Transport = newUserAgentTransport(s.doer.Transport, UserAgent)
	}
}

// New creates a new speedtest client.
func New(opts ...Option) *Speedtest {
	s := &Speedtest{
		doer: http.DefaultClient,
	}
	WithUserAgent(defaultUserAgent)(s)

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func Version() string {
	return version
}

var defaultClient = New()
