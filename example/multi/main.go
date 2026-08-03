package main

import (
	"context"
	"fmt"
	"log"

	"github.com/showwin/speedtest-go/speedtest"
)

func main() {
	serverList, _ := speedtest.FetchServers()
	targets, _ := serverList.FindServer([]int{})

	if len(targets) > 0 {
		// Use s as main server and use targets as auxiliary servers.
		// The main server is loaded at a greater proportion than the auxiliary servers.
		s := targets[0]
		checkError(s.MultiDownloadTestContext(context.TODO(), targets))
		checkError(s.MultiUploadTestContext(context.TODO(), targets))
		fmt.Printf("Download: %s, Upload: %s\n", s.DLSpeed, s.ULSpeed)
	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
