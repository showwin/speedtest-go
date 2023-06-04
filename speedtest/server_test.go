package speedtest

import (
	"testing"
	"time"
)

func TestFetchServerList(t *testing.T) {
	client := New()
	client.User = &User{
		IP:  "111.111.111.111",
		Lat: "35.22",
		Lon: "138.44",
		Isp: "Hello",
	}
	servers, err := client.FetchServers()
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(servers) == 0 {
		t.Errorf("Failed to fetch server list.")
		return
	}
	if len(servers[0].Country) == 0 {
		t.Errorf("got unexpected country name '%v'", servers[0].Country)
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
	servers := Servers{
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

	var serverID []int
	s, err := servers.FindServer(serverID)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(s) != 1 {
		t.Errorf("unexpected server length. got: %v, expected: 1", len(s))
	}
	if s[0].ID != "1" {
		t.Errorf("unexpected server ID. got: %v, expected: '1'", s[0].ID)
	}

	serverID = []int{2}
	s, err = servers.FindServer(serverID)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(s) != 1 {
		t.Errorf("unexpected server length. got: %v, expected: 1", len(s))
	}
	if s[0].ID != "2" {
		t.Errorf("unexpected server ID. got: %v, expected: '2'", s[0].ID)
	}

	serverID = []int{3, 1}
	s, err = servers.FindServer(serverID)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(s) != 2 {
		t.Errorf("unexpected server length. got: %v, expected: 2", len(s))
	}
	if s[0].ID != "3" {
		t.Errorf("unexpected server ID. got: %v, expected: '3'", s[0].ID)
	}
	if s[1].ID != "1" {
		t.Errorf("unexpected server ID. got: %v, expected: '1'", s[0].ID)
	}
}

func TestCustomServer(t *testing.T) {
	// Good server
	got, err := CustomServer("https://example.com/upload.php")
	if err != nil {
		t.Errorf(err.Error())
	}
	if got.Host != "example.com" {
		t.Error("did not properly set the Host field on a custom server")
	}

	// Missing upload.php
	_, err = CustomServer("https://example.com")
	if err == nil {
		t.Error("did not create a custom server without upload.php")
	}
}

func TestTotalDurationCount(t *testing.T) {
	server, _ := CustomServer("https://example.com/upload.php")

	uploadTime := time.Duration(10000805542)
	server.TestDuration.Upload = &uploadTime
	server.testDurationTotalCount()
	if server.TestDuration.Total.Nanoseconds() != 10000805542 {
		t.Error("addition in testDurationTotalCount didn't work")
	}

	downloadTime := time.Duration(10000403875)
	server.TestDuration.Download = &downloadTime
	server.testDurationTotalCount()
	if server.TestDuration.Total.Nanoseconds() != 20001209417 {
		t.Error("addition in testDurationTotalCount do didn't work")
	}

	pingTime := time.Duration(2183156458)
	server.TestDuration.Ping = &pingTime
	server.testDurationTotalCount()
	if server.TestDuration.Total.Nanoseconds() != 22184365875 {
		t.Error("addition in testDurationTotalCount didn't work")
	}
}
