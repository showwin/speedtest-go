package main

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
)

var client = http.Client{}

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

func newTransport(insecure bool, iface string) (tr http.Transport, ip net.IP) {
	tlsConf := &tls.Config{InsecureSkipVerify: insecure}

	if iface != "" {
		ief, err := net.InterfaceByName(iface)
		if err != nil {
			log.Fatal(err)
		}
		addrs, err := ief.Addrs()
		if err != nil {
			log.Fatal(err)
		}

		tcpAddr := &net.TCPAddr{IP: addrs[0].(*net.IPNet).IP}
		d := net.Dialer{LocalAddr: tcpAddr}
		tr = http.Transport{Dial: d.Dial, TLSClientConfig: tlsConf}

		ip = tcpAddr.IP
	} else {
		tr = http.Transport{TLSClientConfig: tlsConf}
	}

	return tr, ip
}

func setSourceAddr(iface string) (ip net.IP) {

	ief, err := net.InterfaceByName(iface)
	if err != nil {
		log.Fatal(err)
	}
	addrs, err := ief.Addrs()
	if err != nil {
		log.Fatal(err)
	}

	tcpAddr := &net.TCPAddr{IP: addrs[0].(*net.IPNet).IP}
	d := net.Dialer{LocalAddr: tcpAddr}

	client.Transport.(*http.Transport).Dial = d.Dial

	return tcpAddr.IP
}

var (
	app        = kingpin.New("speedtest-fpngfw", "Run a speedtest from a Forcepoint NGFW. Writen by Newlode www.newlode.io").Author("Sébastien Boulet @ Newlode Groupe - www.newlode.io")
	insecure   = app.Flag("insecure", "Disable TLS certificate verify").Short('i').Default("true").Bool()
	iface      = app.Flag("iface", "Force the use of IFACE for this test").Short('I').String()
	showList   = app.Flag("list", "Show available speedtest.net servers").Short('l').Bool()
	serverIds  = app.Flag("server", "Select server id to speedtest").Short('s').Ints()
	timeoutOpt = app.Flag("timeout", "Define timeout seconds. Default: 10 sec").Short('t').Int()
	timeout    = 10
)

func main() {

	var ip net.IP
	kingpin.Version("1.0.7")
	kingpin.MustParse(app.Parse(os.Args[1:]))

	setTimeout()
	tr, ip := newTransport(*insecure, *iface)
	client = http.Client{Transport: &tr}

	user := fetchUserInfo()
	user.Show(ip.String())

	list := fetchServerList(user)
	if *showList {
		list.Show()
		return
	}

	targets := list.FindServer(*serverIds)
	targets.StartTest()
	targets.ShowResult()
}
