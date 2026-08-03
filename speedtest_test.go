package main

import (
	"testing"

	"github.com/showwin/speedtest-go/speedtest"
)

func TestFormatUserInfo(t *testing.T) {
	user := &speedtest.User{
		IP:  "219.77.208.183",
		Isp: "Netvigator Home Broadband",
		Lat: "22.2908",
		Lon: "114.1501",
	}

	got := formatUserInfo(user, true)
	want := "***.**.***.*** (Netvigator Home Broadband) [**.****, ***.****]"
	if got != want {
		t.Fatalf("formatUserInfo() = %q, want %q", got, want)
	}
}

func TestMaskSensitiveValue(t *testing.T) {
	got := maskSensitiveValue("2001:db8::1/-22.2908")
	want := "****:***::*/-**.****"
	if got != want {
		t.Fatalf("maskSensitiveValue() = %q, want %q", got, want)
	}
}
