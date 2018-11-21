package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/cbergoon/speedtest-go/speedtest"
)


func setTimeout() {
	if *timeoutOpt != 0 {
		timeout = *timeoutOpt
	}
}

var (
	showList   = kingpin.Flag("list", "Show available speedtest.net servers").Short('l').Bool()
	serverIds  = kingpin.Flag("server", "Select server id to speedtest").Short('s').Ints()
	timeoutOpt = kingpin.Flag("timeout", "Define timeout seconds. Default: 10 sec").Short('t').Int()
	timeout    = 10
)

func main() {
	kingpin.Version("1.0.4")
	kingpin.Parse()

	setTimeout()

	user := speedtest.FetchUserInfo()
	user.Show()

	list := speedtest.FetchServerList(user)
	if *showList {
		list.Show()
		return
	}

	targets := list.FindServer(*serverIds)
	targets.StartTest()
	targets.ShowResult()
}
