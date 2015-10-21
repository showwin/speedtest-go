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

var (
	showList = kingpin.Flag("list", "Show available speedtest.net servers").Short('l').Bool()
	serverIds = kingpin.Flag("server", "Select server id to speedtest").Short('s').Ints()
)

func main() {
	kingpin.Parse()

	user := FetchUserInfo()
	user.Show()

	list := FetchServerList(user)
	if *showList {
		list.Show()
		return
	}

	targets := list.FindServer(*serverIds)
	targets.StartTest()
	targets.ShowResult()
}
