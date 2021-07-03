package speedtest

import "testing"
import "strings"

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
	if len(user.Lat) > 8 {
		t.Errorf("Invalid Latitude. got: %v;", user.Lat)
	}
	if strings.Count(user.Lat, ".") != 1 {
		t.Errorf("Invalid Latitude format. got: %v", user.Lat)
	}

	// Lon
	if len(user.Lon) > 8 {
		t.Errorf("Invalid Londitude. got: %v;", user.Lon)
	}
	if strings.Count(user.Lon, ".") != 1 {
		t.Errorf("Invalid Londitude format. got: %v", user.Lon)
	}

	// Isp
	if len(user.Isp) == 0 {
		t.Errorf("Invalid Iso. got: %v;", user.Isp)
	}
}
