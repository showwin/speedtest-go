package speedtest

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"
)

var (
	version          = "1.5.1"
	DefaultUserAgent = fmt.Sprintf("showwin/speedtest-go %s", version)
)

// Speedtest is a speedtest client.
type Speedtest struct {
	doer      *http.Client
	config    *UserConfig
	tcpDialer *net.Dialer
	ipDialer  *net.Dialer
	Manager
}

type UserConfig struct {
	T         *http.Transport
	UserAgent string
	Proxy     string
	Source    string
	Debug     bool
	ICMP      bool

	SavingMode bool

	CityFlag     string
	LocationFlag string
	Location     *Location

	Keyword string

	NoDownload bool
	NoUpload   bool
}

func parseAddr(addr string) (string, string) {
	prefixIndex := strings.Index(addr, "://")
	if prefixIndex != -1 {
		return addr[:prefixIndex], addr[prefixIndex+3:]
	}
	return "", addr // ignore address network prefix
}

func (s *Speedtest) NewUserConfig(uc *UserConfig) {
	if uc.Debug {
		dbg.Enable()
	}

	if uc.SavingMode {
		s.SetNThread(1) // Set the number of concurrent connections to 1
	}

	if len(uc.CityFlag) > 0 {
		var err error
		uc.Location, err = GetLocation(uc.CityFlag)
		if err != nil {
			dbg.Printf("Warning: skipping command line arguments: --city. err: %v\n", err.Error())
		}
	}
	if len(uc.LocationFlag) > 0 {
		var err error
		uc.Location, err = ParseLocation(uc.CityFlag, uc.LocationFlag)
		if err != nil {
			dbg.Printf("Warning: skipping command line arguments: --location. err: %v\n", err.Error())
		}
	}

	var tcpSource net.Addr // If nil, a local address is automatically chosen.
	var icmpSource net.Addr
	var proxy = http.ProxyFromEnvironment
	s.config = uc
	if len(uc.Source) > 0 {
		_, address := parseAddr(uc.Source)
		addr0, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("[%s]:0", address)) // dynamic tcp port
		if err == nil {
			tcpSource = addr0
		} else {
			dbg.Printf("Warning: skipping parse the source address. err: %s\n", err.Error())
		}
		addr1, err := net.ResolveIPAddr("ip", address) // dynamic tcp port
		if err == nil {
			icmpSource = addr1
		} else {
			dbg.Printf("Warning: skipping parse the source address. err: %s\n", err.Error())
		}
	}

	if len(uc.Proxy) > 0 {
		if parse, err := url.Parse(uc.Proxy); err != nil {
			dbg.Printf("Warning: skipping parse the proxy host. err: %s\n", err.Error())
		} else {
			proxy = func(_ *http.Request) (*url.URL, error) {
				return parse, err
			}
		}
	}

	s.tcpDialer = &net.Dialer{
		LocalAddr: tcpSource,
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	s.ipDialer = &net.Dialer{
		LocalAddr: icmpSource,
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	s.config.T = &http.Transport{
		Proxy:                 proxy,
		DialContext:           s.tcpDialer.DialContext,
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
func WithUserConfig(userConfig *UserConfig) Option {
	return func(s *Speedtest) {
		s.NewUserConfig(userConfig)
		dbg.Printf("Source: %s\n", s.config.Source)
		dbg.Printf("Proxy: %s\n", s.config.Proxy)
		dbg.Printf("SavingMode: %v\n", s.config.SavingMode)
		dbg.Printf("Keyword: %v\n", s.config.Keyword)
		dbg.Printf("ICMP: %v\n", s.config.ICMP)
		dbg.Printf("OS: %s, ARCH: %s, NumCPU: %d\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	}
}

// New creates a new speedtest client.
func New(opts ...Option) *Speedtest {
	log.SetOutput(io.Discard)
	s := &Speedtest{
		doer:    http.DefaultClient,
		Manager: GlobalDataManager,
	}
	// load default config
	s.NewUserConfig(&UserConfig{UserAgent: DefaultUserAgent})

	for _, opt := range opts {
		opt(s)
	}
	return s
}

func Version() string {
	return version
}

var defaultClient = New()
