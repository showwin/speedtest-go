package speedtest

import (
	"strconv"
	"strings"
	"testing"
)

func TestFetchUserInfo(t *testing.T) {
	user, err := FetchUserInfo()
	if err != nil {
		t.Errorf(err.Error())
	}
	// IP
	if len(user.IP) < 7 || len(user.IP) > 15 {
		t.Errorf("Invalid IP length. got: %v;", user.IP)
	}
	if strings.Count(user.IP, ".") != 3 {
		t.Errorf("Invalid IP format. got: %v", user.IP)
	}

	// Lat
	lat, err := strconv.ParseFloat(user.Lat, 64)
	if err != nil {
		t.Errorf(err.Error())
	}
	if lat < -90 || 90 < lat {
		t.Errorf("Invalid Latitude. got: %v, expected between -90 and 90", user.Lat)
	}

	// Lon
	lon, err := strconv.ParseFloat(user.Lon, 64)
	if err != nil {
		t.Errorf(err.Error())
	}
	if lon < -180 || 180 < lon {
		t.Errorf("Invalid Latitude. got: %v, expected between -90 and 90", user.Lon)
	}

	// Isp
	if len(user.Isp) == 0 {
		t.Errorf("Invalid Iso. got: %v;", user.Isp)
	}
}
