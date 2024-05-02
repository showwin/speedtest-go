package main

import (
	"context"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

var (
	showList      = kingpin.Flag("list", "Show available speedtest.net servers.").Short('l').Bool()
	serverIds     = kingpin.Flag("server", "Select server id to run speedtest.").Short('s').Ints()
	customURL     = kingpin.Flag("custom-url", "Specify the url of the server instead of getting a list from speedtest.net.").String()
	savingMode    = kingpin.Flag("saving-mode", "Test with few resources, though low accuracy (especially > 30Mbps).").Bool()
	jsonOutput    = kingpin.Flag("json", "Output results in json format.").Bool()
	location      = kingpin.Flag("location", "Change the location with a precise coordinate. Format: lat,lon").String()
	city          = kingpin.Flag("city", "Change the location with a predefined city label.").String()
	showCityList  = kingpin.Flag("city-list", "List all predefined city labels.").Bool()
	proxy         = kingpin.Flag("proxy", "Set a proxy(http[s] or socks) for the speedtest.").String()
	source        = kingpin.Flag("source", "Bind a source interface for the speedtest.").String()
	dnsBindSource = kingpin.Flag("dns-bind-source", "DNS request binding source.(Experimental)").Bool()
	multi         = kingpin.Flag("multi", "Enable multi-server mode.").Short('m').Bool()
	thread        = kingpin.Flag("thread", "Set the number of concurrent connections.").Short('t').Int()
	search        = kingpin.Flag("search", "Fuzzy search servers by a keyword.").String()
	userAgent     = kingpin.Flag("ua", "Set the user-agent header for the speedtest.").String()
	noDownload    = kingpin.Flag("no-download", "Disable download test.").Bool()
	noUpload      = kingpin.Flag("no-upload", "Disable upload test.").Bool()
	pingMode      = kingpin.Flag("ping-mode", "Select a method for Ping. (support icmp/tcp/http)").Default("http").String()
	debug         = kingpin.Flag("debug", "Enable debug mode.").Short('d').Bool()
)

func main() {

	kingpin.Version(speedtest.Version())
	kingpin.Parse()
	AppInfo()

	// 0. speed test setting
	var speedtestClient = speedtest.New(speedtest.WithUserConfig(
		&speedtest.UserConfig{
			UserAgent:     *userAgent,
			Proxy:         *proxy,
			Source:        *source,
			DnsBindSource: *dnsBindSource,
			Debug:         *debug,
			PingMode:      parseProto(*pingMode), // TCP as default
			SavingMode:    *savingMode,
			CityFlag:      *city,
			LocationFlag:  *location,
			Keyword:       *search,
			NoDownload:    *noDownload,
			NoUpload:      *noUpload,
		}))
	speedtestClient.SetNThread(*thread)

	if *showCityList {
		speedtest.PrintCityList()
		return
	}

	// 1. retrieving user information
	taskManager := InitTaskManager(!*jsonOutput)
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
				task.Printf("Latency: %v", latency)
			}))
			task.Printf("Latency: %v Jitter: %v Min: %v Max: %v", server.Latency, server.Jitter, server.MinLatency, server.MaxLatency)
			task.Complete()
		})
		accEcho := newAccompanyEcho(server, time.Millisecond*500)
		taskManager.Run("Download", func(task *Task) {
			accEcho.Run()
			speedtestClient.SetCallbackDownload(func(downRate float64) {
				lc := accEcho.CurrentLatency()
				if lc == 0 {
					task.Printf("Download: %.2fMbps (latency: --)", downRate*8/1000000)
				} else {
					task.Printf("Download: %.2fMbps (latency: %dms)", downRate*8/1000000, lc/1000000)
				}
			})
			if *multi {
				task.CheckError(server.MultiDownloadTestContext(context.Background(), servers))
			} else {
				task.CheckError(server.DownloadTest())
			}
			accEcho.Stop()
			mean, _, std, minL, maxL := speedtest.StandardDeviation(accEcho.Latencies())
			task.Printf("Download: %.2fMbps (used: %.2fMB) (latency: %dms jitter: %dms min: %dms max: %dms)", server.DLSpeed*8/1000000, float64(server.Context.Manager.GetTotalDownload())/1000/1000, mean/1000000, std/1000000, minL/1000000, maxL/1000000)
			task.Complete()
		})

		taskManager.Run("Upload", func(task *Task) {
			accEcho.Run()
			speedtestClient.SetCallbackUpload(func(upRate float64) {
				lc := accEcho.CurrentLatency()
				if lc == 0 {
					task.Printf("Upload: %.2fMbps (latency: --)", upRate*8/1000000)
				} else {
					task.Printf("Upload: %.2fMbps (latency: %dms)", upRate*8/1000000, lc/1000000)
				}
			})
			if *multi {
				task.CheckError(server.MultiUploadTestContext(context.Background(), servers))
			} else {
				task.CheckError(server.UploadTest())
			}
			accEcho.Stop()
			mean, _, std, minL, maxL := speedtest.StandardDeviation(accEcho.Latencies())
			task.Printf("Upload: %.2fMbps (used: %.2fMB) (latency: %dms jitter: %dms min: %dms max: %dms)", server.ULSpeed*8/1000000, float64(server.Context.Manager.GetTotalUpload())/1000/1000, mean/1000000, std/1000000, minL/1000000, maxL/1000000)
			task.Complete()
		})
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
		fmt.Printf("    speedtest-go v%s @showwin\n", speedtest.Version())
		fmt.Println()
	}
}
