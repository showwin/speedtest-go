package speedtest

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"
)

const speedTestServersUrl = "https://www.speedtest.net/speedtest-servers-static.php"
const speedTestServersAlternativeUrl = "https://www.speedtest.net/speedtest-servers-static.php"

// Server information
type Server struct {
	URL      string        `xml:"url,attr" json:"url"`
	Lat      string        `xml:"lat,attr" json:"lat"`
	Lon      string        `xml:"lon,attr" json:"lon"`
	Name     string        `xml:"name,attr" json:"name"`
	Country  string        `xml:"country,attr" json:"country"`
	Sponsor  string        `xml:"sponsor,attr" json:"sponsor"`
	ID       string        `xml:"id,attr" json:"id"`
	URL2     string        `xml:"url2,attr" json:"url_2"`
	Host     string        `xml:"host,attr" json:"host"`
	Distance float64       `json:"distance"`
	Latency  time.Duration `json:"latency"`
	DLSpeed  float64       `json:"dl_speed"`
	ULSpeed  float64       `json:"ul_speed"`
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
	return FetchServerListContext(context.Background(), user)
}

// FetchServerListContext retrieves a list of available servers, observing the given context.
func FetchServerListContext(ctx context.Context, user *User) (ServerList, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, speedTestServersUrl, nil)
	if err != nil {
		return ServerList{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ServerList{}, err
	}

	if resp.ContentLength == 0 {
		resp.Body.Close()

		req, err = http.NewRequestWithContext(ctx, http.MethodGet, speedTestServersAlternativeUrl, nil)
		if err != nil {
			return ServerList{}, err
		}

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return ServerList{}, err
		}
	}

	defer resp.Body.Close()

	// Decode xml
	decoder := xml.NewDecoder(resp.Body)

	var list ServerList
	if err := decoder.Decode(&list); err != nil {
		return list, err
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
