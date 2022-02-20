package speedtest

type Location struct {
	Lat float64
	Lon float64
}

// Locations TODO more location need to added
var Locations = map[string]Location{
	"hongkong":     {Lat: 22.3207, Lon: 114.1689},
	"chiyoda":      {Lat: 35.6869, Lon: 139.7575},
	"london":       {Lat: 51.5063, Lon: -0.1201},
	"moscow":       {Lat: 55.7520, Lon: 37.6175},
	"beijing":      {Lat: 39.5600, Lon: 116.2000},
	"paris":        {Lat: 48.8600, Lon: 2.3390},
	"sanfrancisco": {Lat: 37.7687, Lon: -122.4754},
}
