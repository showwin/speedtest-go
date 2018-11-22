package main

import (
	"gopkg.in/alecthomas/kingpin.v2"

	"log"
	"os"
	"fmt"
	"time"
	"github.com/cbergoon/speedtest-go/speedtest"
)

var (
	showList  = kingpin.Flag("list", "Show available speedtest.net servers").Short('l').Bool()
	serverIds = kingpin.Flag("server", "Select server id to speedtest").Short('s').Ints()
	//timeoutOpt = kingpin.Flag("timeout", "Define timeout seconds. Default: 10 sec").Short('t').Int()
)

func main() {
	kingpin.Version("1.0.3")
	kingpin.Parse()

	user, err := speedtest.FetchUserInfo()
	if err != nil {
		fmt.Println("Warning: Cannot fetch user information. http://www.speedtest.net/speedtest-config.php is temporarily unavailable.")
	}
	showUser(user)

	serverList, err := speedtest.FetchServerList(user)
	checkError(err)
	if *showList {
		showServerList(serverList)
		return
	}

	targets, err := serverList.FindServer(*serverIds)
	checkError(err)

	startTest(targets)
}

func startTest(servers speedtest.Servers) {
	for _, s := range servers {
		showServer(s)

		err := s.PingTest()
		checkError(err)
		showLatencyResult(s)

		err = testDownload(s)
		checkError(err)
		err = testUpload(s)
		checkError(err)

		showServerResult(s)
	}

	if len(servers) > 1 {
		showAverageServerResult(servers)
	}
}

func testDownload(server *speedtest.Server) error {
	quit := make(chan bool)
	fmt.Printf("Download Test: ")
	go dots(quit)
	err := server.DownloadTest()
	quit <- true
	if err != nil {
		return err
	}
	fmt.Println()
	return err
}

func testUpload(server *speedtest.Server) error {
	quit := make(chan bool)
	fmt.Printf("Upload Test: ")
	go dots(quit)
	err := server.UploadTest()
	quit <- true
	if err != nil {
		return err
	}
	fmt.Println()
	return nil
}

func dots(quit chan bool) {
	for {
		select {
		case <-quit:
			return
		default:
			time.Sleep(time.Second)
			fmt.Print(".")
		}
	}
}

func showUser(user *speedtest.User) {
	if user.IP != "" {
		fmt.Printf("Testing From IP: %s\n", user.String())
	}
}

func showServerList(serverList speedtest.ServerList) {
	for _, s := range serverList.Servers {
		fmt.Printf("[%4s] %8.2fkm ", s.ID, s.Distance)
		fmt.Printf(s.Name + " (" + s.Country + ") by " + s.Sponsor + "\n")
	}
}

func showServer(s *speedtest.Server) {
	fmt.Printf(" \n")
	fmt.Printf("Target Server: [%4s] %8.2fkm ", s.ID, s.Distance)
	fmt.Printf(s.Name + " (" + s.Country + ") by " + s.Sponsor + "\n")
}

func showLatencyResult(server *speedtest.Server) {
	fmt.Println("Latency:", server.Latency)
}

// ShowResult : show testing result
func showServerResult(server *speedtest.Server) {
	fmt.Printf(" \n")

	fmt.Printf("Download: %5.2f Mbit/s\n", server.DLSpeed)
	fmt.Printf("Upload: %5.2f Mbit/s\n\n", server.ULSpeed)
	valid := server.CheckResultValid()
	if !valid {
		fmt.Println("Warning: Result seems to be wrong. Please speedtest again.")
	}
}

func showAverageServerResult(servers speedtest.Servers) {
	avgDL := 0.0
	avgUL := 0.0
	for _, s := range servers {
		avgDL = avgDL + s.DLSpeed
		avgUL = avgUL + s.ULSpeed
	}
	fmt.Printf("Download Avg: %5.2f Mbit/s\n", avgDL/float64(len(servers)))
	fmt.Printf("Upload Avg: %5.2f Mbit/s\n", avgUL/float64(len(servers)))
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
