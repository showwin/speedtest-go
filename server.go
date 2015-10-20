package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

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

func FetchServerList(user User) List {
	// fetch xml server data
	resp, err := http.Get("http://www.speedtest.net/speedtest-servers-static.php")
	CheckError(err)
	body, err := ioutil.ReadAll(resp.Body)
	CheckError(err)
	defer resp.Body.Close()

	// decode xml
	decoder := xml.NewDecoder(bytes.NewReader(body))
	list := List{}
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
		sLat, _ := strconv.ParseFloat(server.Lat, 64)
		sLon, _ := strconv.ParseFloat(server.Lon, 64)
		uLat, _ := strconv.ParseFloat(user.Lat, 64)
		uLon, _ := strconv.ParseFloat(user.Lon, 64)
		server.Distance = Distance(sLat, sLon, uLat, uLon)
	}

	// sort by distance
	sort.Sort(ByDistance{list.Servers})

	return list
}

func Distance(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
	radius := 6378.137

	a1 := lat1 * math.Pi / 180.0
	b1 := lon1 * math.Pi / 180.0
	a2 := lat2 * math.Pi / 180.0
	b2 := lon2 * math.Pi / 180.0

	x := math.Sin(a1)*math.Sin(a2) + math.Cos(a1)*math.Cos(a2)*math.Cos(b2-b1)
	return radius * math.Acos(x)
}

func (l List) FindServer(serverId int) Server {
	// default
	if serverId == 0 {
		return l.Servers[1]
	}

	// --server option
	for _, s := range l.Servers {
		sid, _ := strconv.Atoi(s.Id)
		if serverId == sid {
			return s
		}
	}
	return l.Servers[1]
}

func (l *List) Show() {
	for _, s := range l.Servers {
		fmt.Printf("[%4s] %8.2fkm ", s.Id, s.Distance)
		fmt.Printf(s.Name + " (" + s.Country + ") by " + s.Sponsor + "\n")
	}
}

func (s Server) Show() {
	fmt.Printf("Target Server: [%4s] %8.2fkm ", s.Id, s.Distance)
	fmt.Printf(s.Name + " (" + s.Country + ") by " + s.Sponsor + "\n")
}

func (s Server) DownloadTest() float64 {
	dlUrl := strings.Split(s.Url, "/upload")[0]
	return DownloadSpeed(dlUrl)
}

func (s Server) UploadTest() float64 {
	return UploadSpeed(s.Url)
}
