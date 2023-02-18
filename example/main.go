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

	serverList, _ := speedtest.FetchServers(user)
	targets, _ := serverList.FindServer([]int{})

	for _, s := range targets {
		checkError(s.PingTest())
		checkError(s.DownloadTest(false))
		checkError(s.UploadTest(false))

		fmt.Printf("Latency: %s, Download: %f, Upload: %f\n", s.Latency, s.DLSpeed, s.ULSpeed)
		speedtest.GlobalDataManager.Reset()
	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
