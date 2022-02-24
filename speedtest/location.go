package speedtest

import "fmt"

type Location struct {
	Lat float64
	Lon float64
}

// Locations TODO more location need to added
var Locations = map[string]Location{
	"br-brasilia":     {-15.793876, -47.8835327},
	"cn-hongkong":     {22.3207, 114.1689},
	"jp-tokyo":        {35.6869, 139.7575},
	"uk-london":       {51.5063, -0.1201},
	"ru-moscow":       {55.7520, 37.6175},
	"cn-beijing":      {39.8721243, 116.4077473},
	"fr-paris":        {48.8600, 2.3390},
	"us-sanfrancisco": {37.7687, -122.4754},
	"us-newyork":      {40.7200876, -74.0220945},
	"sg-yishun":       {1.4230218, 103.8404728},
	"in-delhi":        {28.6251287, 77.1960896},
}

func PrintCityList() {
	fmt.Println("Available city labels (case insensitive): ")
	for k, v := range Locations {
		fmt.Printf("%s -> %v\n", k, v)
	}
}
