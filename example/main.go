package main

import (
	"fmt"
	"github.com/showwin/speedtest-go/speedtest"
	"log"
)

func main() {
	user, _ := speedtest.FetchUserInfo()
	// Get a list of servers near a specified location
	// user.SetLocationByCity("Tokyo")
	// user.SetLocation("Osaka", 34.6952, 135.5006)

	// Select a network card as the data interface.
	// speedtest.WithUserConfig(&speedtest.UserConfig{Source: "tcp4://192.168.1.101"})(speedtestClient)

	serverList, _ := speedtest.FetchServers(user)
	targets, _ := serverList.FindServer([]int{})

	for _, s := range targets {
		// Please make sure your host can access this test server,
		// otherwise you will get an error.
		// It is recommended to replace a server at this time
		checkError(s.PingTest())
		checkError(s.DownloadTest(false))
		checkError(s.UploadTest(false))

		fmt.Printf("Latency: %s, Download: %f, Upload: %f\n", s.Latency, s.DLSpeed, s.ULSpeed)
		//speedtest.GlobalDataManager.Reset()
	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
