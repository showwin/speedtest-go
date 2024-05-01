package main

import (
	"fmt"
	"github.com/showwin/speedtest-go/speedtest"
	"syscall"
)

func customControl(network string, address string, conn syscall.RawConn) error {
	fmt.Printf("%s,%s\n", network, address)
	return nil
}

func main() {
	var speedtestClient = speedtest.New()
	serverList, _ := speedtestClient.FetchServers()
	targets, _ := serverList.FindServer([]int{})

	var d []float64

	speedtestClient.CallbackDownloadRate(func(downRate float64) {
		d = append(d, downRate)
	})

	for _, s := range targets {
		// Please make sure your host can access this test server,
		// otherwise you will get an error.
		// It is recommended to replace a server at this time
		s.DownloadTest()
		fmt.Printf("Latency: %s, Download: %f, Upload: %f\n", s.Latency, s.DLSpeed, s.ULSpeed)
		s.Context.Reset() // reset counter
	}

	fmt.Print("data := []float64{")
	for _, val := range d {
		fmt.Printf("%v, ", val)
	}
	fmt.Print("}")
}
