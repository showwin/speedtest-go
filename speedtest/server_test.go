package speedtest

import (
	"fmt"
	"reflect"
	"testing"
)

func TestFetchServerList(t *testing.T) {
	user := User{
		IP:  "111.111.111.111",
		Lat: "35.22",
		Lon: "138.44",
		Isp: "Hello",
	}
	serverList, err := FetchServerList(&user)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(serverList.Servers) == 0 {
		t.Errorf("Failed to fetch server list.")
	}
	if len(serverList.Servers[0].Country) == 0 {
		t.Errorf("got unexpected country name '%v'", serverList.Servers[0].Country)
	}
}

func extractServerID(servers []*Server) []string {
	serverID := []string{}
	for _, server := range servers {
		serverID = append(serverID, server.ID)
	}
	return serverID
}

func TestFetchServerListManyTimes(t *testing.T) {
	user := User{
		IP:  "111.111.111.111",
		Lat: "35.22",
		Lon: "138.44",
		Isp: "Hello",
	}
	firstResult, _ := FetchServerList(&user)
	firstServerIDs := extractServerID(firstResult.Servers)

	for i := 1; i <= 100; i++ {
		result, _ := FetchServerList(&user)
		serverIDs := extractServerID(result.Servers)
		if !reflect.DeepEqual(firstServerIDs, serverIDs) {
			fmt.Println(firstServerIDs)
			fmt.Printf("=============\n")
			fmt.Println(serverIDs)
			t.Errorf("Server list is different from each request.")
		}
	}
}

func TestDistance(t *testing.T) {
	d := distance(0.0, 0.0, 1.0, 1.0)
	if d < 157 || 158 < d {
		t.Errorf("got: %v, expected between 157 and 158", d)
	}

	d = distance(0.0, 180.0, 0.0, -180.0)
	if d != 0 {
		t.Errorf("got: %v, expected 0", d)
	}

	d1 := distance(100.0, 100.0, 100.0, 101.0)
	d2 := distance(100.0, 100.0, 100.0, 99.0)
	if d1 != d2 {
		t.Errorf("%v and %v should be save value", d1, d2)
	}

	d = distance(35.0, 140.0, -40.0, -140.0)
	if d < 11000 || 12000 < d {
		t.Errorf("got: %v, expected 0", d)
	}
}

func TestFindServer(t *testing.T) {
	servers := []*Server{
		&Server{
			ID: "1",
		},
		&Server{
			ID: "2",
		},
		&Server{
			ID: "3",
		},
	}
	serverList := ServerList{Servers: servers}

	serverID := []int{}
	s, err := serverList.FindServer(serverID)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(s) != 1 {
		t.Errorf("Unexpected server length. got: %v, expected: 1", len(s))
	}
	if s[0].ID != "1" {
		t.Errorf("Unexpected server ID. got: %v, expected: '1'", s[0].ID)
	}

	serverID = []int{2}
	s, err = serverList.FindServer(serverID)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(s) != 1 {
		t.Errorf("Unexpected server length. got: %v, expected: 1", len(s))
	}
	if s[0].ID != "2" {
		t.Errorf("Unexpected server ID. got: %v, expected: '2'", s[0].ID)
	}

	serverID = []int{3, 1}
	s, err = serverList.FindServer(serverID)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(s) != 2 {
		t.Errorf("Unexpected server length. got: %v, expected: 2", len(s))
	}
	if s[0].ID != "3" {
		t.Errorf("Unexpected server ID. got: %v, expected: '3'", s[0].ID)
	}
	if s[1].ID != "1" {
		t.Errorf("Unexpected server ID. got: %v, expected: '1'", s[0].ID)
	}
}
