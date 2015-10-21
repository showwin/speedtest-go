package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
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

func FetchUserInfo() User {
	// fetch xml user data
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
	return users.Users[0]
}

func (u *User) Show() {
	fmt.Println("IP: " + u.Ip + " (" + u.Isp + ") [" + u.Lat + ", " + u.Lon + "]")
}
