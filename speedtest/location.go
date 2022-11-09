package speedtest

import "fmt"

type Location struct {
	CC  string
	Lat float64
	Lon float64
}

// Locations TODO more location need to added
var Locations = map[string]Location{
	"brasilia":     {"br", -15.793876, -47.8835327},
	"hongkong":     {"hk", 22.3106806, 114.1700546},
	"tokyo":        {"jp", 35.680938, 139.7674114},
	"london":       {"uk", 51.5072493, -0.1288861},
	"moscow":       {"ru", 55.7497248, 37.615989},
	"beijing":      {"cn", 39.8721243, 116.4077473},
	"paris":        {"fr", 48.8626026, 2.3477229},
	"sanfrancisco": {"us", 37.7540028, -122.4429967},
	"newyork":      {"us", 40.7200876, -74.0220945},
	"yishun":       {"sg", 1.4230218, 103.8404728},
	"delhi":        {"in", 28.6251287, 77.1960896},
	"monterrey":    {"mx", 25.6881435, -100.3073485},
	"berlin":       {"de", 52.5168128, 13.4009469},
	"maputo":       {"mz", -25.9579267, 32.5760444},
	"honolulu":     {"us", 20.8247065, -156.918706},
	"seoul":        {"kr", 37.6086268, 126.7179721},
	"osaka":        {"jp", 34.6952743, 135.5006967},
	"shanghai":     {"cn", 31.2292105, 121.4661666},
	"urumqi":       {"cn", 43.8256624, 87.6058564},
	"ottawa":       {"ca", 45.4161836, -75.7035467},
	"capetown":     {"za", -33.9391993, 18.4316716},
	"sydney":       {"au", -33.8966622, 151.1731861},
	"perth":        {"au", -31.9551812, 115.8591904},
	"warsaw":       {"pl", 52.2396659, 21.0129345},
	"kampala":      {"ug", 0.3070027, 32.5675581},
	"bangkok":      {"th", 13.7248936, 100.493026},
}

func PrintCityList() {
	fmt.Println("Available city labels (case insensitive): ")
	fmt.Println(" CC\t\tCityLabel\tLocation")
	for k, v := range Locations {
		fmt.Printf("(%v)\t%20s\t[%v, %v]\n", v.CC, k, v.Lat, v.Lon)
	}
}
