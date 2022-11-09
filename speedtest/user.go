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

// SetLocationByCity set current location using predefined location label.
func (u *User) SetLocationByCity(locationLabel string) (err error) {
	loc, ok := Locations[strings.ToLower(locationLabel)]
	if ok {
		u.SetLocation(locationLabel, loc.Lat, loc.Lon)
	} else {
		err = fmt.Errorf("no found predefined label: %s", locationLabel)
	}
	return
}

// SetLocation set the latitude and longitude of the current user
func (u *User) SetLocation(locationName string, latitude float64, longitude float64) {
	u.VLat = fmt.Sprintf("%v", latitude)
	u.VLon = fmt.Sprintf("%v", longitude)
	u.VLoc = strings.Title(locationName)
}

// ParseAndSetLocation parse latitude and longitude string
func (u *User) ParseAndSetLocation(coordinateStr string) error {
	ll := strings.Split(coordinateStr, ",")
	if len(ll) == 2 {
		// parameter validity check
		lat, err := betweenRange(ll[0], 90)
		if err != nil {
			return err
		}
		lon, err := betweenRange(ll[1], 180)
		if err != nil {
			return err
		}

		u.SetLocation("Customize", lat, lon)
		return nil
	}
	return fmt.Errorf("invalid location input: %s", coordinateStr)
}

// betweenRange latitude and longitude range check
func betweenRange(inputStrValue string, interval float64) (float64, error) {
	value, err := strconv.ParseFloat(inputStrValue, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid input: %v", inputStrValue)
	}
	if value < -interval || interval < value {
		return 0, fmt.Errorf("invalid input. got: %v, expected between -%v and %v", inputStrValue, interval, interval)
	}
	return value, nil
}
