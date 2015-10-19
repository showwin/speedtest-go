package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
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

// for sort
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

func FetchUserInfo() {
	// fetch xml user data
	fmt.Println("Retrieving User Information ...")
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
	fmt.Println("Retrieving Server Information ...")
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
		fmt.Printf("[%4s] %8.2fkm "+server.Name+" ", server.Id, server.Distance)
		fmt.Printf("(" + server.Country + ") by " + server.Sponsor + "\n")
	}
}

func CheckError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var list = List{}
var user = User{}
var showList = kingpin.Flag("list", "show available speedtest.net servers").Short('l').Bool()

func main() {
	kingpin.Parse()

	FetchUserInfo()
	ShowUserInfo()
	FetchServerList()
	if *showList {
		ShowServerList()
	}
}
