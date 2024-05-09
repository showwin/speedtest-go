package main

import (
	"fmt"
	"github.com/showwin/speedtest-go/speedtest"
	"github.com/showwin/speedtest-go/speedtest/transport"
	"log"
	"sync"
	"time"
)

// Note: The current packet loss analyzer does not support udp over http.
// This means we cannot get packet loss through a proxy.
func main() {
	// 0. Fetching servers
	serverList, err := speedtest.FetchServers()
	checkError(err)

	// 1. Retrieve available servers
	targets := serverList.Available()

	// 2. Create a packet loss analyzer, use default options
	analyzer := speedtest.NewPacketLossAnalyzer(&speedtest.PacketLossAnalyzerOptions{
		PacketSendingInterval: time.Millisecond * 100,
	})

	wg := &sync.WaitGroup{}
	// 3. Perform packet loss analysis on all available servers
	var hosts []string
	for _, server := range *targets {
		hosts = append(hosts, server.Host)
		wg.Add(1)
		//ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
		//go func(server *speedtest.Server, analyzer *speedtest.PacketLossAnalyzer, ctx context.Context, cancel context.CancelFunc) {
		go func(server *speedtest.Server, analyzer *speedtest.PacketLossAnalyzer) {
			//defer cancel()
			defer wg.Done()
			// Note: Please call ctx.cancel at the appropriate time to release resources if you use analyzer.RunWithContext
			// we using analyzer.Run() here.
			err = analyzer.Run(server.Host, func(packetLoss *transport.PLoss) {
				fmt.Println(packetLoss, server.Host, server.Name)
			})
			//err = analyzer.RunWithContext(ctx, server.Host, func(packetLoss *transport.PLoss) {
			//	fmt.Println(packetLoss, server.Host, server.Name)
			//})
			if err != nil {
				fmt.Println(err)
			}
			//}(server, analyzer, ctx, cancel)
		}(server, analyzer)
	}
	wg.Wait()

	// use mixed PacketLoss
	mixed, err := analyzer.RunMulti(hosts)
	checkError(err)
	fmt.Printf("Mixed packets lossed: %.2f\n", mixed)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
