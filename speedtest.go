package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/showwin/speedtest-go/speedtest/transport"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/showwin/speedtest-go/speedtest"
)

var (
	showList      = kingpin.Flag("list", "Show available speedtest.net servers.").Short('l').Bool()
	serverIds     = kingpin.Flag("server", "Select server id to run speedtest.").Short('s').Ints()
	customURL     = kingpin.Flag("custom-url", "Specify the url of the server instead of fetching from speedtest.net.").String()
	savingMode    = kingpin.Flag("saving-mode", "Test with few resources, though low accuracy (especially > 30Mbps).").Bool()
	jsonOutput    = kingpin.Flag("json", "Output results in json format.").Bool()
	unixOutput    = kingpin.Flag("unix", "Output results in unix like format.").Bool()
	location      = kingpin.Flag("location", "Change the location with a precise coordinate (format: lat,lon).").String()
	city          = kingpin.Flag("city", "Change the location with a predefined city label.").String()
	showCityList  = kingpin.Flag("city-list", "List all predefined city labels.").Bool()
	proxy         = kingpin.Flag("proxy", "Set a proxy(http[s] or socks) for the speedtest.").String()
	source        = kingpin.Flag("source", "Bind a source interface for the speedtest.").String()
	dnsBindSource = kingpin.Flag("dns-bind-source", "DNS request binding source (experimental).").Bool()
	multi         = kingpin.Flag("multi", "Enable multi-server mode.").Short('m').Bool()
	thread        = kingpin.Flag("thread", "Set the number of concurrent connections.").Short('t').Int()
	search        = kingpin.Flag("search", "Fuzzy search servers by a keyword.").String()
	userAgent     = kingpin.Flag("ua", "Set the user-agent header for the speedtest.").String()
	noDownload    = kingpin.Flag("no-download", "Disable download test.").Bool()
	noUpload      = kingpin.Flag("no-upload", "Disable upload test.").Bool()
	pingMode      = kingpin.Flag("ping-mode", "Select a method for Ping (support icmp/tcp/http).").Default("http").String()
	unit          = kingpin.Flag("unit", "Set human-readable and auto-scaled rate units for output (options: decimal-bits/decimal-bytes/binary-bits/binary-bytes).").Short('u').String()
	debug         = kingpin.Flag("debug", "Enable debug mode.").Short('d').Bool()
	dlSize        = kingpin.Flag("dl-size", "Set download chunk size.").Default("1000").Int()
	ulSize        = kingpin.Flag("ul-size", "Set upload chunk size.").Default("800").Int()
)

var (
	commit = "dev"
	date   = "unknown"
)

func main() {
	kingpin.Version(fmt.Sprintf("speedtest-go v%s git-%s built at %s", speedtest.Version(), commit, date))
	kingpin.Parse()
	AppInfo()

	speedtest.SetUnit(parseUnit(*unit))

	// discard standard log.
	log.SetOutput(io.Discard)

	// start unix output for saving mode by default.
	if *savingMode && !*jsonOutput && !*unixOutput {
		*unixOutput = true
	}

	// 0. speed test setting
	var speedtestClient = speedtest.New(speedtest.WithUserConfig(
		&speedtest.UserConfig{
			UserAgent:      *userAgent,
			Proxy:          *proxy,
			Source:         *source,
			DnsBindSource:  *dnsBindSource,
			Debug:          *debug,
			PingMode:       parseProto(*pingMode), // TCP as default
			SavingMode:     *savingMode,
			MaxConnections: *thread,
			CityFlag:       *city,
			LocationFlag:   *location,
			Keyword:        *search,
			DlSize:         *dlSize,
			UlSize:         *ulSize,
		}))

	if *showCityList {
		speedtest.PrintCityList()
		return
	}

	// 1. retrieving user information
	taskManager := InitTaskManager(*jsonOutput, *unixOutput)
	taskManager.AsyncRun("Retrieving User Information", func(task *Task) {
		u, err := speedtestClient.FetchUserInfo()
		task.CheckError(err)
		task.Printf("ISP: %s", u.String())
		task.Complete()
	})

	// 2. retrieving servers
	var err error
	var servers speedtest.Servers
	var targets speedtest.Servers
	taskManager.Run("Retrieving Servers", func(task *Task) {
		if len(*customURL) > 0 {
			var target *speedtest.Server
			target, err = speedtestClient.CustomServer(*customURL)
			task.CheckError(err)
			targets = []*speedtest.Server{target}
			task.Println("Skip: Using Custom Server")
		} else if len(*serverIds) > 0 {
			// TODO: need async fetch to speedup
			for _, id := range *serverIds {
				serverPtr, errFetch := speedtestClient.FetchServerByID(strconv.Itoa(id))
				if errFetch != nil {
					continue // Silently Skip all ids that actually don't exist.
				}
				targets = append(targets, serverPtr)
			}
			task.CheckError(err)
			task.Printf("Found %d Specified Public Server(s)", len(targets))
		} else {
			servers, err = speedtestClient.FetchServers()
			task.CheckError(err)
			task.Printf("Found %d Public Servers", len(servers))
			if *showList {
				task.Complete()
				task.manager.Reset()
				showServerList(servers)
				os.Exit(0)
			}
			targets, err = servers.FindServer(*serverIds)
			task.CheckError(err)
		}
		task.Complete()
	})
	taskManager.Reset()

	// 3. test each selected server with ping, download and upload.
	for _, server := range targets {
		if !*jsonOutput {
			fmt.Println()
		}
		taskManager.Println("Test Server: " + server.String())
		taskManager.Run("Latency: --", func(task *Task) {
			task.CheckError(server.PingTest(func(latency time.Duration) {
				task.Updatef("Latency: %v", latency)
			}))
			task.Printf("Latency: %v Jitter: %v Min: %v Max: %v", server.Latency, server.Jitter, server.MinLatency, server.MaxLatency)
			task.Complete()
		})

		// 3.0 create a packet loss analyzer, use default options
		analyzer := speedtest.NewPacketLossAnalyzer(&speedtest.PacketLossAnalyzerOptions{
			SourceInterface: *source,
		})

		blocker := sync.WaitGroup{}
		packetLossAnalyzerCtx, packetLossAnalyzerCancel := context.WithTimeout(context.Background(), time.Second*40)
		taskManager.Run("Packet Loss Analyzer", func(task *Task) {
			blocker.Add(1)
			go func() {
				defer blocker.Done()
				err = analyzer.RunWithContext(packetLossAnalyzerCtx, server.Host, func(packetLoss *transport.PLoss) {
					server.PacketLoss = *packetLoss
				})
				if errors.Is(err, transport.ErrUnsupported) {
					packetLossAnalyzerCancel() // cancel early
				}
			}()
			task.Println("Packet Loss Analyzer: Running in background (<= 30 Secs)")
			task.Complete()
		})

		// 3.1 create accompany Echo
		accEcho := newAccompanyEcho(server, time.Millisecond*500)
		taskManager.RunWithTrigger(!*noDownload, "Download", func(task *Task) {
			accEcho.Run()
			speedtestClient.SetCallbackDownload(func(downRate speedtest.ByteRate) {
				lc := accEcho.CurrentLatency()
				if lc == 0 {
					task.Updatef("Download: %s (Latency: --)", downRate)
				} else {
					task.Updatef("Download: %s (Latency: %dms)", downRate, lc/1000000)
				}
			})
			if *multi {
				task.CheckError(server.MultiDownloadTestContext(context.Background(), servers))
			} else {
				task.CheckError(server.DownloadTest())
			}
			accEcho.Stop()
			mean, _, std, minL, maxL := speedtest.StandardDeviation(accEcho.Latencies())
			task.Printf("Download: %s (Used: %.2fMB) (Latency: %dms Jitter: %dms Min: %dms Max: %dms)", server.DLSpeed, float64(server.Context.Manager.GetTotalDownload())/1000/1000, mean/1000000, std/1000000, minL/1000000, maxL/1000000)
			task.Complete()
		})

		taskManager.RunWithTrigger(!*noUpload, "Upload", func(task *Task) {
			accEcho.Run()
			speedtestClient.SetCallbackUpload(func(upRate speedtest.ByteRate) {
				lc := accEcho.CurrentLatency()
				if lc == 0 {
					task.Updatef("Upload: %s (Latency: --)", upRate)
				} else {
					task.Updatef("Upload: %s (Latency: %dms)", upRate, lc/1000000)
				}
			})
			if *multi {
				task.CheckError(server.MultiUploadTestContext(context.Background(), servers))
			} else {
				task.CheckError(server.UploadTest())
			}
			accEcho.Stop()
			mean, _, std, minL, maxL := speedtest.StandardDeviation(accEcho.Latencies())
			task.Printf("Upload: %s (Used: %.2fMB) (Latency: %dms Jitter: %dms Min: %dms Max: %dms)", server.ULSpeed, float64(server.Context.Manager.GetTotalUpload())/1000/1000, mean/1000000, std/1000000, minL/1000000, maxL/1000000)
			task.Complete()
		})

		if *noUpload && *noDownload {
			time.Sleep(time.Second * 30)
		}
		packetLossAnalyzerCancel()
		blocker.Wait()
		if !*jsonOutput {
			taskManager.Println(server.PacketLoss.String())
		}
		taskManager.Reset()
		speedtestClient.Manager.Reset()
	}
	taskManager.Stop()

	if *jsonOutput {
		json, errMarshal := speedtestClient.JSON(targets)
		if errMarshal != nil {
			panic(errMarshal)
		}
		fmt.Print(string(json))
	}
}

type AccompanyEcho struct {
	stopEcho       chan bool
	server         *speedtest.Server
	currentLatency int64
	interval       time.Duration
	latencies      []int64
}

func newAccompanyEcho(server *speedtest.Server, interval time.Duration) *AccompanyEcho {
	return &AccompanyEcho{
		server:   server,
		interval: interval,
		stopEcho: make(chan bool),
	}
}

func (ae *AccompanyEcho) Run() {
	ae.latencies = make([]int64, 0)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-ae.stopEcho:
				cancel()
				return
			default:
				latency, _ := ae.server.HTTPPing(ctx, 1, ae.interval, nil)
				if len(latency) > 0 {
					atomic.StoreInt64(&ae.currentLatency, latency[0])
					ae.latencies = append(ae.latencies, latency[0])
				}
			}
		}
	}()
}

func (ae *AccompanyEcho) Stop() {
	ae.stopEcho <- false
}

func (ae *AccompanyEcho) CurrentLatency() int64 {
	return atomic.LoadInt64(&ae.currentLatency)
}

func (ae *AccompanyEcho) Latencies() []int64 {
	return ae.latencies
}

func showServerList(servers speedtest.Servers) {
	for _, s := range servers {
		fmt.Printf("[%5s] %9.2fkm ", s.ID, s.Distance)

		if s.Latency == -1 {
			fmt.Printf("%v", "Timeout ")
		} else {
			fmt.Printf("%-dms ", s.Latency/time.Millisecond)
		}
		fmt.Printf("\t%s (%s) by %s \n", s.Name, s.Country, s.Sponsor)
	}
}

func parseUnit(str string) speedtest.UnitType {
	str = strings.ToLower(str)
	if str == "decimal-bits" {
		return speedtest.UnitTypeDecimalBits
	} else if str == "decimal-bytes" {
		return speedtest.UnitTypeDecimalBytes
	} else if str == "binary-bits" {
		return speedtest.UnitTypeBinaryBits
	} else if str == "binary-bytes" {
		return speedtest.UnitTypeBinaryBytes
	} else {
		return speedtest.UnitTypeDefaultMbps
	}
}

func parseProto(str string) speedtest.Proto {
	str = strings.ToLower(str)
	if str == "icmp" {
		return speedtest.ICMP
	} else if str == "tcp" {
		return speedtest.TCP
	} else {
		return speedtest.HTTP
	}
}

func AppInfo() {
	if !*jsonOutput {
		fmt.Println()
		fmt.Printf("    speedtest-go v%s (git-%s) @showwin\n", speedtest.Version(), commit)
		fmt.Println()
	}
}
