package main

import (
	"log"
	"os"
	"net/http"
	"crypto/tls"

	"gopkg.in/alecthomas/kingpin.v2"
)

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func setTimeout() {
	if *timeoutOpt != 0 {
		timeout = *timeoutOpt
	}
}

func setCertVerify(insecure bool) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: insecure,
	}
}

var (
	insecure   = kingpin.Flag("insecure", "Disable TLS certificate verify").Short('i').Bool()
	showList   = kingpin.Flag("list", "Show available speedtest.net servers").Short('l').Bool()
	serverIds  = kingpin.Flag("server", "Select server id to speedtest").Short('s').Ints()
	timeoutOpt = kingpin.Flag("timeout", "Define timeout seconds. Default: 10 sec").Short('t').Int()
	timeout    = 10
)

func main() {
	kingpin.Version("1.0.3")
	kingpin.Parse()

	setTimeout()
	setCertVerify(*insecure)

	user := fetchUserInfo()
	user.Show()

	list := fetchServerList(user)
	if *showList {
		list.Show()
		return
	}

	targets := list.FindServer(*serverIds)
	targets.StartTest()
	targets.ShowResult()
}
