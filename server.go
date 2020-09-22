package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"sort"
	"strconv"

	"github.com/logrusorgru/aurora"
)

// Server information
type Server struct {
	URL      string `xml:"url,attr"`
	Lat      string `xml:"lat,attr"`
	Lon      string `xml:"lon,attr"`
	Name     string `xml:"name,attr"`
	Country  string `xml:"country,attr"`
	Sponsor  string `xml:"sponsor,attr"`
	ID       string `xml:"id,attr"`
	URL2     string `xml:"url2,attr"`
	Host     string `xml:"host,attr"`
	Distance float64
	DLSpeed  float64
	ULSpeed  float64
}

// ServerList : List of Server
type ServerList struct {
	Servers []Server `xml:"servers>server"`
}

// Servers : For sorting servers.
type Servers []Server

// ByDistance : For sorting servers.
type ByDistance struct {
	Servers
}

// Len : length of servers. For sorting servers.
func (s Servers) Len() int {
	return len(s)
}

// Swap : swap i-th and j-th. For sorting servers.
func (s Servers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less : compare the distance. For sorting servers.
func (b ByDistance) Less(i, j int) bool {
	return b.Servers[i].Distance < b.Servers[j].Distance
}

func fetchServerList(user User) ServerList {
	// Fetch xml server data
	resp, err := client.Get("http://www.speedtest.net/speedtest-servers-static.php")
	checkError(err)
	body, err := ioutil.ReadAll(resp.Body)
	checkError(err)
	defer resp.Body.Close()

	if len(body) == 0 {
		resp, err = client.Get("http://c.speedtest.net/speedtest-servers-static.php")
		checkError(err)
		body, err = ioutil.ReadAll(resp.Body)
		checkError(err)
		defer resp.Body.Close()
	}

	// Decode xml
	decoder := xml.NewDecoder(bytes.NewReader(body))
	list := ServerList{}
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

	// Calculate distance
	for i := range list.Servers {
		server := &list.Servers[i]
		sLat, _ := strconv.ParseFloat(server.Lat, 64)
		sLon, _ := strconv.ParseFloat(server.Lon, 64)
		uLat, _ := strconv.ParseFloat(user.Lat, 64)
		uLon, _ := strconv.ParseFloat(user.Lon, 64)
		server.Distance = distance(sLat, sLon, uLat, uLon)
	}

	// Sort by distance
	sort.Sort(ByDistance{list.Servers})

	return list
}

func distance(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
	radius := 6378.137

	a1 := lat1 * math.Pi / 180.0
	b1 := lon1 * math.Pi / 180.0
	a2 := lat2 * math.Pi / 180.0
	b2 := lon2 * math.Pi / 180.0

	x := math.Sin(a1)*math.Sin(a2) + math.Cos(a1)*math.Cos(a2)*math.Cos(b2-b1)
	return radius * math.Acos(x)
}

// FindServer : find server by serverID
func (l *ServerList) FindServer(serverID []int) Servers {
	servers := Servers{}

	for _, sid := range serverID {
		for _, s := range l.Servers {
			id, _ := strconv.Atoi(s.ID)
			if sid == id {
				servers = append(servers, s)
			}
		}
	}

	if len(servers) == 0 {
		servers = append(servers, l.Servers[0])
	}

	return servers
}

// Show : show server list
func (l ServerList) Show() {
	for _, s := range l.Servers {
		fmt.Printf("[%4s] %8.2fkm ", s.ID, s.Distance)
		fmt.Printf(s.Name + " (" + s.Country + ") by " + s.Sponsor + "\n")
	}
}

// Show : show server information
func (s Server) Show() {
	fmt.Printf("%-6s : %s %.2fkm (%s/%s) by %s\n", "Target", aurora.Magenta(s.ID), s.Distance, aurora.Magenta(s.Name), s.Country, aurora.Gray(12, s.Sponsor))
}

// StartTest : start testing to the servers.
func (svrs Servers) StartTest() {
	for i, s := range svrs {
		s.Show()
		latency := pingTest(s.URL)
		dlSpeed := downloadTest(s.URL, latency)
		ulSpeed := uploadTest(s.URL, latency)
		svrs[i].DLSpeed = dlSpeed
		svrs[i].ULSpeed = ulSpeed
	}
}

// ShowResult : show testing result
func (svrs Servers) ShowResult() {
	fmt.Printf(" \n")
	if len(svrs) == 1 {
		fmt.Printf("%-13s : %s\n", "Download", aurora.Gray(24, fmt.Sprintf("%5.2f Mbit/s", svrs[0].DLSpeed)))
		fmt.Printf("%-13s : %s\n", "Upload", aurora.Gray(24, fmt.Sprintf("%5.2f Mbit/s", svrs[0].ULSpeed)))
	} else {
		for _, s := range svrs {
			fmt.Printf("[%4s] Download: %5.2f Mbit/s, Upload: %5.2f Mbit/s\n", s.ID, s.DLSpeed, s.ULSpeed)
		}
		avgDL := 0.0
		avgUL := 0.0
		for _, s := range svrs {
			avgDL = avgDL + s.DLSpeed
			avgUL = avgUL + s.ULSpeed
		}
		fmt.Printf("Download Avg: %5.2f Mbit/s\n", avgDL/float64(len(svrs)))
		fmt.Printf("Upload Avg: %5.2f Mbit/s\n", avgUL/float64(len(svrs)))
	}
	err := svrs.checkResult()
	if err {
		fmt.Println("Warning: Result seems to be wrong. Please speedtest again.")
	}
}

func (svrs Servers) checkResult() bool {
	errFlg := false
	if len(svrs) == 1 {
		s := svrs[0]
		errFlg = (s.DLSpeed*100 < s.ULSpeed) || (s.DLSpeed > s.ULSpeed*100)
	} else {
		for _, s := range svrs {
			errFlg = errFlg || (s.DLSpeed*100 < s.ULSpeed) || (s.DLSpeed > s.ULSpeed*100)
		}
	}
	return errFlg
}
