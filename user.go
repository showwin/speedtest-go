package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"

	"github.com/logrusorgru/aurora"
)

// User information
type User struct {
	IP  string `xml:"ip,attr"`
	Lat string `xml:"lat,attr"`
	Lon string `xml:"lon,attr"`
	Isp string `xml:"isp,attr"`
}

// Users : for decode xml
type Users struct {
	Users []User `xml:"client"`
}

func fetchUserInfo() User {
	// Fetch xml user data
	resp, err := client.Get("http://speedtest.net/speedtest-config.php")
	checkError(err)
	body, err := ioutil.ReadAll(resp.Body)
	checkError(err)
	defer resp.Body.Close()

	// Decode xml
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
	if users.Users == nil {
		fmt.Println("Warning: Cannot fetch user information. http://www.speedtest.net/speedtest-config.php is temporarily unavailable.")
		return User{}
	}
	return users.Users[0]
}

// Show user location
func (u *User) Show(ip string) {
	if u.IP != "" {
		fmt.Printf("%-6s : %s/%s (%s) [%s,%s]\n", "Source", aurora.Gray(18, ip), aurora.Gray(24, u.IP), aurora.Cyan(u.Isp), aurora.Gray(12, u.Lat), aurora.Gray(12, u.Lon))
	}
}
