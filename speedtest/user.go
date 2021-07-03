package speedtest

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
)

const speedTestConfigUrl = "https://www.speedtest.net/speedtest-config.php"

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
	return FetchUserInfoContext(context.Background())
}

// FetchUserInfoContext returns information about caller determined by speedtest.net, observing the given context.
func FetchUserInfoContext(ctx context.Context) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, speedTestConfigUrl, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Decode xml
	decoder := xml.NewDecoder(resp.Body)

	var users Users
	if err := decoder.Decode(&users); err != nil {
		return nil, err
	}

	if len(users.Users) == 0 {
		return nil, errors.New("failed to fetch user information")
	}

	return &users.Users[0], nil
}

// String representation of User
func (u *User) String() string {
	return fmt.Sprintf("%s, (%s) [%s, %s]", u.IP, u.Isp, u.Lat, u.Lon)
}
