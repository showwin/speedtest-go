package speedtest

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"sort"
	"strconv"
	"errors"
	"time"
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
	Latency  time.Duration
	DLSpeed  float64
	ULSpeed  float64
}

// ServerList list of Server
type ServerList struct {
	Servers []*Server `xml:"servers>server"`
}

// Servers for sorting servers.
type Servers []*Server

// ByDistance for sorting servers.
type ByDistance struct {
	Servers
}

// Len finds length of servers. For sorting servers.
func (svrs Servers) Len() int {
	return len(svrs)
}

// Swap swaps i-th and j-th. For sorting servers.
func (svrs Servers) Swap(i, j int) {
	svrs[i], svrs[j] = svrs[j], svrs[i]
}

// Less compares the distance. For sorting servers.
func (b ByDistance) Less(i, j int) bool {
	return b.Servers[i].Distance < b.Servers[j].Distance
}

// FetchServerList retrieves a list of available servers
func FetchServerList(user *User) (ServerList, error) {
	// Fetch xml server data
	resp, err := http.Get("http://www.speedtest.net/speedtest-servers-static.php")
	if err != nil {
		return ServerList{}, errors.New("failed to retrieve speedtest servers")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ServerList{}, errors.New("failed to read response body")
	}
	defer resp.Body.Close()

	if len(body) == 0 {
		resp, err = http.Get("http://c.speedtest.net/speedtest-servers-static.php")
		if err != nil {
			errors.New("failed to retrieve alternate speedtest servers")
		}
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return ServerList{}, errors.New("failed to read response body")
		}
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
		server := list.Servers[i]
		sLat, _ := strconv.ParseFloat(server.Lat, 64)
		sLon, _ := strconv.ParseFloat(server.Lon, 64)
		uLat, _ := strconv.ParseFloat(user.Lat, 64)
		uLon, _ := strconv.ParseFloat(user.Lon, 64)
		server.Distance = distance(sLat, sLon, uLat, uLon)
	}

	// Sort by distance
	sort.Sort(ByDistance{list.Servers})

	if len(list.Servers) <= 0 {
		return list, errors.New("unable to retrieve server list")
	}

	return list, nil
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

// FindServer finds server by serverID
func (l *ServerList) FindServer(serverID []int) (Servers, error) {
	servers := Servers{}

	if len(l.Servers) <= 0 {
		return servers, errors.New("no servers available")
	}

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

	return servers, nil
}

// String representation of ServerList
func (l *ServerList) String() string {
	slr := ""
	for _, s := range l.Servers {
		slr += s.String()
	}
	return slr
}

// String representation of Server
func (s *Server) String() string {
	return fmt.Sprintf("[%4s] %8.2fkm \n%s (%s) by %s\n", s.ID, s.Distance, s.Name, s.Country, s.Sponsor)
}

// CheckResultValid checks that results are logical given UL and DL speeds
func (s Server) CheckResultValid() bool {
	return !(s.DLSpeed*100 < s.ULSpeed) || !(s.DLSpeed > s.ULSpeed*100)
}
