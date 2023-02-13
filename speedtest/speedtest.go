package speedtest

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	version          = "1.3.2"
	DefaultUserAgent = fmt.Sprintf("showwin/speedtest-go %s", version)
)

// Speedtest is a speedtest client.
type Speedtest struct {
	doer *http.Client

	config *UserConfig
}

type UserConfig struct {
	T         *http.Transport
	UserAgent string
	Proxy     string
	OutBound  string
}

func parseAddr(addr string) (string, string) {
	prefixIndex := strings.Index(addr, "://")
	if prefixIndex != -1 {
		return addr[:prefixIndex], addr[prefixIndex+3:]
	}
	return "", addr // ignore address network prefix
}

func (s *Speedtest) NewUserConfig(uc *UserConfig) {
	var outbound *net.TCPAddr // If nil, a local address is automatically chosen.
	var proxy = http.ProxyFromEnvironment

	s.config = uc

	if len(uc.OutBound) > 0 {
		network, address := parseAddr(uc.OutBound)
		addr, err := net.ResolveTCPAddr(network, fmt.Sprintf("%s:0", address)) // dynamic tcp port
		if err == nil {
			outbound = addr
		} else {
			log.Printf("Skip: can not parse the outbound address. err: %s\n", err.Error())
		}
	}

	if len(uc.Proxy) > 0 {
		if parse, err := url.Parse(uc.Proxy); err != nil {
			log.Printf("Skip: can not parse the proxy host. err: %s\n", err.Error())
		} else {
			proxy = func(_ *http.Request) (*url.URL, error) {
				//return url.Parse(uc.Proxy)
				return parse, err
			}
		}
	}

	dialer := net.Dialer{
		LocalAddr: outbound,
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	s.config.T = &http.Transport{
		Proxy:                 proxy,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	s.doer.Transport = s
}

func (s *Speedtest) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", s.config.UserAgent)
	return s.config.T.RoundTrip(req)
}

// Option is a function that can be passed to New to modify the Client.
type Option func(*Speedtest)

// WithDoer sets the http.Client used to make requests.
func WithDoer(doer *http.Client) Option {
	return func(s *Speedtest) {
		s.doer = doer
	}
}

// WithUserConfig adds a custom user config for speedtest.
// This configuration may be overwritten again by WithDoer,
// because client and transport are parent-child relationship:
// `New(WithDoer(myDoer), WithUserAgent(myUserAgent), WithDoer(myDoer))`
func WithUserConfig(userAgent *UserConfig) Option {
	return func(s *Speedtest) {
		s.NewUserConfig(userAgent)
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

func Version() string {
	return version
}

var defaultClient = New(WithUserConfig(&UserConfig{UserAgent: DefaultUserAgent}))
