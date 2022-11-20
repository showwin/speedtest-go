package speedtest

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"
)

const speedTestServersUrl = "https://www.speedtest.net/api/js/servers?limit=10"
const speedTestServersAlternativeUrl = "https://www.speedtest.net/speedtest-servers-static.php"

type PayloadType int

const (
	JSONPayload PayloadType = iota
	XMLPayload
)

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

	doer *http.Client
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

// FetchServers retrieves a list of available servers
func (client *Speedtest) FetchServers(user *User) (Servers, error) {
	return client.FetchServerListContext(context.Background(), user)
}

// FetchServers retrieves a list of available servers
func FetchServers(user *User) (Servers, error) {
	return defaultClient.FetchServers(user)
}

// FetchServerListContext retrieves a list of available servers, observing the given context.
func (client *Speedtest) FetchServerListContext(ctx context.Context, user *User) (Servers, error) {
	fetchUrl := fmt.Sprintf("%s&lat=%s&lon=%s", speedTestServersUrl, user.VLat, user.VLon)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchUrl, nil)
	if err != nil {
		return Servers{}, err
	}

	resp, err := client.doer.Do(req)
	if err != nil {
		return Servers{}, err
	}

	payloadType := JSONPayload

	if resp.ContentLength == 0 {
		resp.Body.Close()

		req, err = http.NewRequestWithContext(ctx, http.MethodGet, speedTestServersAlternativeUrl, nil)
		if err != nil {
			return Servers{}, err
		}

		resp, err = client.doer.Do(req)
		if err != nil {
			return Servers{}, err
		}

		payloadType = XMLPayload
	}

	defer resp.Body.Close()

	var servers Servers

	switch payloadType {
	case JSONPayload:
		// Decode xml
		decoder := json.NewDecoder(resp.Body)

		if err := decoder.Decode(&servers); err != nil {
			return servers, err
		}
	case XMLPayload:
		var list ServerList
		// Decode xml
		decoder := xml.NewDecoder(resp.Body)

		if err := decoder.Decode(&list); err != nil {
			return servers, err
		}

		servers = list.Servers
	default:
		return servers, fmt.Errorf("response payload decoding not implemented")
	}

	// set doer of server
	for _, s := range servers {
		s.doer = client.doer
	}

	// Calculate distance
	for _, server := range servers {
		sLat, _ := strconv.ParseFloat(server.Lat, 64)
		sLon, _ := strconv.ParseFloat(server.Lon, 64)
		uLat, _ := strconv.ParseFloat(user.Lat, 64)
		uLon, _ := strconv.ParseFloat(user.Lon, 64)
		server.Distance = distance(sLat, sLon, uLat, uLon)
	}

	// Sort by distance
	sort.Sort(ByDistance{servers})

	if len(servers) <= 0 {
		return servers, errors.New("unable to retrieve server list")
	}

	return servers, nil
}

// FetchServerListContext retrieves a list of available servers, observing the given context.
func FetchServerListContext(ctx context.Context, user *User) (Servers, error) {
	return defaultClient.FetchServerListContext(ctx, user)
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
func (l Servers) FindServer(serverID []int) (Servers, error) {
	servers := Servers{}

	if len(l) <= 0 {
		return servers, errors.New("no servers available")
	}

	for _, sid := range serverID {
		for _, s := range l {
			id, _ := strconv.Atoi(s.ID)
			if sid == id {
				servers = append(servers, s)
			}
		}
	}

	if len(servers) == 0 {
		servers = append(servers, l[0])
	}

	return servers, nil
}

// String representation of ServerList
func (l ServerList) String() string {
	slr := ""
	for _, s := range l.Servers {
		slr += s.String()
	}
	return slr
}

// String representation of Servers
func (l Servers) String() string {
	slr := ""
	for _, s := range l {
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
