package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/cheggaaa/pb"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type User struct {
	Ip  string `xml:"ip,attr"`
	Lat string `xml:"lat,attr"`
	Lon string `xml:"lon,attr"`
	Isp string `xml:"isp,attr"`
}

type Users struct {
	Users []User `xml:"client"`
}

type Server struct {
	Url      string `xml:"url,attr"`
	Lat      string `xml:"lat,attr"`
	Lon      string `xml:"lon,attr"`
	Name     string `xml:"name,attr"`
	Country  string `xml:"country,attr"`
	Sponsor  string `xml:"sponsor,attr"`
	Id       string `xml:"id,attr"`
	Url2     string `xml:"url2,attr"`
	Host     string `xml:"host,attr"`
	Distance float64
}

type List struct {
	Servers []Server `xml:"servers>server"`
}

// for sort =start=
type Servers []Server

type ByDistance struct {
	Servers
}

func (s Servers) Len() int {
	return len(s)
}

func (s Servers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (b ByDistance) Less(i, j int) bool {
	return b.Servers[i].Distance < b.Servers[j].Distance
}

// for sort =end=

func FetchUserInfo() {
	// fetch xml user data
	resp, err := http.Get("http://www.speedtest.net/speedtest-config.php")
	CheckError(err)
	body, err := ioutil.ReadAll(resp.Body)
	CheckError(err)
	defer resp.Body.Close()

	// decode xml
	decoder := xml.NewDecoder(bytes.NewReader(body))
	users := Users{}
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			decoder.DecodeElement(&users, &se)
		}
	}
	user = users.Users[0]
}

func FetchServerList() {
	// fetch xml server data
	resp, err := http.Get("http://www.speedtest.net/speedtest-servers-static.php")
	CheckError(err)
	body, err := ioutil.ReadAll(resp.Body)
	CheckError(err)
	defer resp.Body.Close()

	// decode xml
	decoder := xml.NewDecoder(bytes.NewReader(body))
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			decoder.DecodeElement(&list, &se)
		}
	}

	// calculate distance
	for i := range list.Servers {
		server := &list.Servers[i]
		lat, _ := strconv.ParseFloat(server.Lat, 64)
		lon, _ := strconv.ParseFloat(server.Lon, 64)
		server.Distance = Distance(lat, lon)
	}

	// sort by distance
	sort.Sort(ByDistance{list.Servers})
}

func Distance(lat float64, lon float64) float64 {
	radius := 6378.137

	lat1 := lat * math.Pi / 180.0
	lon1 := lon * math.Pi / 180.0
	user_lat, _ := strconv.ParseFloat(user.Lat, 64)
	user_lon, _ := strconv.ParseFloat(user.Lon, 64)
	lat2 := user_lat * math.Pi / 180.0
	lon2 := user_lon * math.Pi / 180.0

	x := math.Sin(lat1)*math.Sin(lat2) + math.Cos(lat1)*math.Cos(lat2)*math.Cos(lon2-lon1)
	return radius * math.Acos(x)
}

func ShowUserInfo() {
	fmt.Println("IP: " + user.Ip + " (" + user.Isp + ") [" + user.Lat + ", " + user.Lon + "]")
}

func ShowServerList() {
	for _, server := range list.Servers {
		fmt.Printf("[%4s] %8.2fkm ", server.Id, server.Distance)
		fmt.Printf(server.Name + " (" + server.Country + ") by " + server.Sponsor + "\n")
	}
}

func CheckError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func FindServer() {
	// default
	if *serverId == 0 {
		testServer = list.Servers[1]
		return
	}

	// --server option
	for _, server := range list.Servers {
		sid, _ := strconv.Atoi(server.Id)
		if *serverId == sid {
			testServer = server
		}
	}
	if testServer.Url == "" {
		testServer = list.Servers[1]
	}
}

func ShowServerInfo() {
	fmt.Printf("Target Server: [%4s] %8.2fkm ", testServer.Id, testServer.Distance)
	fmt.Printf(testServer.Name + " (" + testServer.Country + ") by " + testServer.Sponsor + "\n")
}

func DownloadTest() {
	dlUrl := strings.Split(testServer.Url, "/upload")[0]
	dlSpeed = DownloadSpeed(dlUrl)
}

func DownloadSpeed(dlUrl string) float64 {
	fmt.Println("Testing Download Speed ...")
	count := 40 * (40 + 1) / 2
	bar := pb.StartNew(count)
	bar.ShowBar = false
	bar.ShowCounters = false
	sizes := [...]int{350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000}
	urls := [40]string{}
	for i, size := range sizes {
		for j := 0; j < 4; j++ {
			urls[i*4+j] = dlUrl + "/random" + strconv.Itoa(size) + "x" + strconv.Itoa(size) + ".jpg"
		}
	}

	totalTime := time.Duration(0)
	for i, url := range urls {
		for j := 0; j <= i; j++ {
			bar.Increment()
		}
		start_time := time.Now()
		resp, err := http.Get(url)
		CheckError(err)
		ioutil.ReadAll(resp.Body)
		finish_time := time.Now()
		defer resp.Body.Close()

		totalTime = totalTime + finish_time.Sub(start_time)
	}

	sumSize := 0.0
	for _, size := range sizes {
		sumSize = sumSize + 4*2*float64(size)*float64(size)/1000/1000
	}

	return sumSize * 8 / totalTime.Seconds()
}

func UploadTest() {
	ulSpeed = UploadSpeed(testServer.Url)
}

func UploadSpeed(ulUrl string) float64 {
	fmt.Println("Testing Upload Speed ...")
	count := 40 * (40 + 1) / 2
	bar := pb.StartNew(count)
	bar.ShowBar = false
	bar.ShowCounters = false
	sizes := [...]int{100, 300, 500, 800, 1000, 2000, 3000, 4000} //kB

	testSizes := [40]int{}
	for i, size := range sizes {
		for j := 0; j < 5; j++ {
			testSizes[i*5+j] = size
		}
	}

	sumSize := 0
	totalTime := time.Duration(0)
	for i, size := range testSizes {
		for j := 0; j <= i; j++ {
			bar.Increment()
		}
		v := url.Values{}
		v.Add("content", strings.Repeat("0", size*1000 - 160))

		start_time := time.Now()
		resp, err := http.PostForm(ulUrl, v)
		CheckError(err)
		r_body, _ := ioutil.ReadAll(resp.Body)
		finish_time := time.Now()
		defer resp.Body.Close()

		totalTime = totalTime + finish_time.Sub(start_time)
		s, _ := strconv.Atoi(string(r_body)[5:])
		sumSize = sumSize + s
	}

	return float64(sumSize) * 8 / 1000 / 1000 / totalTime.Seconds()
}

func ShowResult() {
	fmt.Printf("Download: %5.2f Mbit/s\n", dlSpeed)
	fmt.Printf("Upload: %5.2f Mbit/s\n", ulSpeed)
}

var list = List{}
var user = User{}
var testServer = Server{}
var showList = kingpin.Flag("list", "show available speedtest.net servers").Short('l').Bool()
var serverId = kingpin.Flag("server", "select server id to speedtest").Short('s').Int()
var dlSpeed = 0.0
var ulSpeed = 0.0

func main() {
	kingpin.Parse()

	FetchUserInfo()
	ShowUserInfo()
	FetchServerList()
	if *showList {
		ShowServerList()
		return
	}
	FindServer()
	ShowServerInfo()
	DownloadTest()
	UploadTest()
	ShowResult()
}
