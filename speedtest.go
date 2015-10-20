package main

import (
	"fmt"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
)

func CheckError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func ShowResult(dlSpeed float64, ulSpeed float64) {
	fmt.Printf("Download: %5.2f Mbit/s\n", dlSpeed)
	fmt.Printf("Upload: %5.2f Mbit/s\n", ulSpeed)
}

var showList = kingpin.Flag("list", "Show available speedtest.net servers").Short('l').Bool()
var serverId = kingpin.Flag("server", "Select server id to speedtest").Short('s').Int()

func main() {
	kingpin.Parse()

	user := FetchUserInfo()
	user.Show()

	list := FetchServerList(user)
	if *showList {
		list.Show()
		return
	}
	target := list.FindServer(*serverId)
	target.Show()
	dlSpeed := target.DownloadTest()
	ulSpeed := target.UploadTest()
	ShowResult(dlSpeed, ulSpeed)
}
