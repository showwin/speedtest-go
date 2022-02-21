package speedtest

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const speedTestConfigUrl = "https://www.speedtest.net/speedtest-config.php"

// User represents information determined about the caller by speedtest.net
type User struct {
	IP   string `xml:"ip,attr"`
	Lat  string `xml:"lat,attr"`
	Lon  string `xml:"lon,attr"`
	Isp  string `xml:"isp,attr"`
	VLoc string `xml:"v_loc,attr"` // virtual location name
	VLat string `xml:"v_lat,attr"` // virtual lat
	VLon string `xml:"v_lon,attr"` // virtual lon
}

// Users for decode xml
type Users struct {
	Users []User `xml:"client"`
}

// FetchUserInfo returns information about caller determined by speedtest.net
func (client *Speedtest) FetchUserInfo() (*User, error) {
	return client.FetchUserInfoContext(context.Background())
}

// FetchUserInfo returns information about caller determined by speedtest.net
func FetchUserInfo() (*User, error) {
	return defaultClient.FetchUserInfo()
}

// FetchUserInfoContext returns information about caller determined by speedtest.net, observing the given context.
func (client *Speedtest) FetchUserInfoContext(ctx context.Context) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, speedTestConfigUrl, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.doer.Do(req)
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

// FetchUserInfoContext returns information about caller determined by speedtest.net, observing the given context.
func FetchUserInfoContext(ctx context.Context) (*User, error) {
	return defaultClient.FetchUserInfoContext(ctx)
}

// String representation of User
func (u *User) String() string {
	extInfo := ""
	if u.VLon != "" {
		extInfo = fmt.Sprintf("-> (%s) [%s, %s]", u.VLoc, u.VLat, u.VLon)
	}
	return fmt.Sprintf("%s, (%s) [%s, %s] %s", u.IP, u.Isp, u.Lat, u.Lon, extInfo)
}

func (u *User) Location(inputLocationName string) (err error) {
	loc, ok := Locations[strings.ToLower(inputLocationName)]
	if ok {
		u.VLat = fmt.Sprintf("%.4f", loc.Lat)
		u.VLon = fmt.Sprintf("%.4f", loc.Lon)
		u.VLoc = strings.Title(inputLocationName)
	} else {
		err = u.parseLocation(inputLocationName)
	}
	return
}

func (u *User) parseLocation(sLoc string) error {
	ll := strings.Split(sLoc, ",")
	if len(ll) == 2 {
		// parameter validity check
		err := betweenRange(ll[0], 90)
		if err != nil {
			return err
		}
		err = betweenRange(ll[1], 180)
		if err != nil {
			return err
		}

		u.VLat = ll[0]
		u.VLon = ll[1]
		u.VLoc = "Customize"
		return nil
	}
	return errors.New("Warning: no found predefined or invalid custom location: " + sLoc)
}

func betweenRange(inputStrValue string, interval float64) error {
	value, err := strconv.ParseFloat(inputStrValue, 64)
	if err != nil {
		return errors.New(fmt.Sprintf("Warning: invalid input: %v", inputStrValue))
	}
	if value < -interval || interval < value {
		return errors.New(fmt.Sprintf("Warning: invalid input. got: %v, expected between -%v and %v", inputStrValue, interval, interval))
	}
	return nil
}
