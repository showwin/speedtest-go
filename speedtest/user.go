package speedtest

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"errors"
)

// User represents information determined about the caller by speedtest.net
type User struct {
	IP  string `xml:"ip,attr"`
	Lat string `xml:"lat,attr"`
	Lon string `xml:"lon,attr"`
	Isp string `xml:"isp,attr"`
}

// Users for decode xml
type Users struct {
	Users []User `xml:"client"`
}

// FetchUserInfo returns information about caller determined by speedtest.net
func FetchUserInfo() (*User, error) {
	// Fetch xml user data
	resp, err := http.Get("http://speedtest.net/speedtest-config.php")
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
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
		return nil, errors.New("failed to fetch user information")
	}
	return &users.Users[0], nil
}

// String representation of User
func (u *User) String() string {
	return fmt.Sprintf("%s, (%s) [%s, %s]", u.IP, u.Isp, u.Lat, u.Lon)
}
