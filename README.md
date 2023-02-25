# speedtest-go
**Command Line Interface and pure Go API to Test Internet Speed using [speedtest.net](http://www.speedtest.net/)**.

You can speedtest 2x faster than [speedtest.net](http://www.speedtest.net/) with almost the same result. [See the experimental results.](https://github.com/showwin/speedtest-go#summary-of-experimental-results).
Inspired by [sivel/speedtest-cli](https://github.com/sivel/speedtest-cli)

Go API Installation below.

## CLI
### Installation
#### macOS (homebrew)

```bash
$ brew tap showwin/speedtest
$ brew install speedtest

### How to Update ###
$ brew update
$ brew upgrade speedtest
```

#### Other Platform (Linux, Windows, etc.)

Please download compatible package from [Releases](https://github.com/showwin/speedtest-go/releases).
If there are no compatible package you want, please let me know on [issue](https://github.com/showwin/speedtest-go/issues).

### Usage

```bash
$ speedtest --help
usage: speedtest-go [<flags>]

Flags:
      --help               Show context-sensitive help (also try --help-long and --help-man).
  -l, --list               Show available speedtest.net servers.
  -s, --server=SERVER ...  Select server id to speedtest.
      --custom-url=CUSTOM-URL Specify the url of the server instead of getting a list from Speedtest.net
      --saving-mode        Using less memory (≒10MB), though low accuracy (especially > 30Mbps).
      --json               Output results in json format
      --location=LOCATION  Change the location with a precise coordinate.
      --city=CITY          Change the location with a predefined city label.
      --city-list          List all predefined city label.
      --proxy              Set a proxy(http(s) or socks) for the speedtest.
                           eg: --proxy=socks://10.20.0.101:7890
                           eg: --proxy=http://10.20.0.101:7890
      --source             Bind a source interface for the speedtest.
                           eg: --source=10.20.0.101
  -m  --multi              Enable multi mode.
  -t  --thread             Set the number of speedtest threads.
      --version            Show application version.
```

#### Test Internet Speed

Simply use `speedtest` command. The closest server is selected by default.

```bash
$ speedtest
Testing From IP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]

Target Server: [6691]     9.03km Shizuoka (Japan) by sudosan
Latency: 24.15396ms
Jitter: 777.465µs
Min: 22.8926ms
Max: 25.5387ms
Download Test: ................
Upload Test: ................

Download: 73.30 Mbit/s
Upload: 35.26 Mbit/s
```

#### Test to Other Servers

If you want to select other server to test, you can see available server list.

```bash
$ speedtest --list
Testing From IP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]
[6691]     9.03km Shizuoka (Japan) by sudosan
[6087]   120.55km Fussa-shi (Japan) by Allied Telesis Capital Corporation
[6508]   125.44km Yokohama (Japan) by at2wn
[6424]   148.23km Tokyo (Japan) by Cordeos Corp.
...
```

and select them by id.

```bash
$ speedtest --server 6691 --server 6087
Testing From IP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]

Target Server: [6691]     9.03km Shizuoka (Japan) by sudosan
Latency: 21.424ms
Jitter: 1.644ms
Min: 19.142ms
Max: 23.926ms
Download Test: ................
Upload Test: ........

Target Server: [6087]   120.55km Fussa-shi (Japan) by Allied Telesis Capital Corporation
Latency: 38.694699ms
Jitter: 2.724ms
Min: 36.443ms
Max: 39.953ms
Download Test: ................
Upload Test: ................

[6691] Download: 65.82 Mbit/s, Upload: 27.00 Mbit/s
[6087] Download: 72.24 Mbit/s, Upload: 29.56 Mbit/s
Download Avg: 69.03 Mbit/s
Upload Avg: 28.28 Mbit/s
```

#### Test with virtual location

With `--city` or `--location` option, the closest server of the location will be picked.
You can measure the speed between your location and the target location.

```bash
$ speedtest --city-list
Available city labels (case insensitive):
 CC             CityLabel       Location
(za)                capetown    [-33.9391993, 18.4316716]
(pl)                  warsaw    [52.2396659, 21.0129345]
(sg)                  yishun    [1.4230218, 103.8404728]
...

$ speedtest --city=capetown
$ speedtest --location=60,-110
```

#### Memory Saving Mode

With `--saving-mode` option, it can be executed even in insufficient memory environment like IoT device.
The memory usage can be reduced to 1/10, about 10MB of memory is used.

However, please be careful that the accuracy is particularly low especially in an environment of 30 Mbps or higher.
To get more accurate results, run multiple times and average.

For more details, please see [saving mode experimental result](https://github.com/showwin/speedtest-go/blob/master/docs/saving_mode_experimental_result.md).

⚠️This feature has been deprecated > v1.4.0, because speedtest-go can always run with less than 10MByte of memory now.

## Go API

```
go get github.com/showwin/speedtest-go
```

### API Usage

The [code](https://github.com/showwin/speedtest-go/blob/master/example/main.go) below finds the closest available speedtest server and tests the latency, download, and upload speeds.
```go
package main

import (
	"fmt"
	"github.com/showwin/speedtest-go/speedtest"
)

func main() {
	var speedtestClient = speedtest.New()
	
	// Use a proxy for the speedtest. eg: socks://127.0.0.1:7890
	// speedtest.WithUserConfig(&speedtest.UserConfig{Proxy: "socks://127.0.0.1:7890"})(speedtestClient)
	
	// Select a network card as the data interface.
	// speedtest.WithUserConfig(&speedtest.UserConfig{Source: "192.168.1.101"})(speedtestClient)
	
	user, _ := speedtestClient.FetchUserInfo()
	// Get a list of servers near a specified location
	// user.SetLocationByCity("Tokyo")
	// user.SetLocation("Osaka", 34.6952, 135.5006)

	serverList, _ := speedtestClient.FetchServers(user)
	targets, _ := serverList.FindServer([]int{})

	for _, s := range targets {
		// Please make sure your host can access this test server,
		// otherwise you will get an error.
		// It is recommended to replace a server at this time
		s.PingTest()
		s.DownloadTest(false)
		s.UploadTest(false)
		fmt.Printf("Latency: %s, Download: %f, Upload: %f\n", s.Latency, s.DLSpeed, s.ULSpeed)
	}
}
```


## Summary of Experimental Results

Speedtest-go is a great tool because of following 4 reasons:
* Cross-platform available.
* Low memory environment.
* Testing time is the **SHORTEST** compare to [speedtest.net](http://www.speedtest.net/) and [sivel/speedtest-cli](https://github.com/sivel/speedtest-cli), especially about 2x faster then [speedtest.net](http://www.speedtest.net/).
* Result is **MORE CLOSE** to [speedtest.net](http://www.speedtest.net/) than [speedtest-cli](https://github.com/sivel/speedtest-cli).

Following data is summarized. If you got interested in, please see [more details](https://github.com/showwin/speedtest-go/blob/master/docs/experimental_result.md).

### Download (Mbps)

distance = distance to testing server
* 0 - 1000(km) ≒ domestic
* 1000 - 8000(km) ≒ same region
* 8000 - 20000(km) ≒ really far!
* 20000km is the half of the circumference of our planet.

| distance (km) | speedtest.net | speedtest-go | speedtest-cli |
|:-------------:|:-------------:|:------------:|:-------------:|
|   0 - 1000    |     92.12     |  **91.21**   |     70.27     |
|  1000 - 8000  |     66.45     |  **65.51**   |     56.56     |
| 8000 - 20000  |     11.84     |     9.43     |   **11.87**   |

### Upload (Mbps)

| distance (km) | speedtest.net | speedtest-go | speedtest-cli |
|:-------------:|:-------------:|:------------:|:-------------:|
|   0 - 1000    |     65.56     |  **47.58**   |     36.16     |
|  1000 - 8000  |     58.02     |  **54.74**   |     26.78     |
| 8000 - 20000  |     5.20      |   **8.32**   |     2.58      |

### Testing Time (sec)

| distance (km) | speedtest.net | speedtest-go | speedtest-cli |
|:-------------:|:-------------:|:------------:|:-------------:|
|   0 - 1000    |     45.03     |  **22.84**   |     24.46     |
|  1000 - 8000  |     44.89     |  **24.45**   |     28.52     |
| 8000 - 20000  |     49.64     |  **34.08**   |     41.26     |

## Contributors

https://github.com/showwin/speedtest-go/graphs/contributors

## LICENSE

[MIT](https://github.com/showwin/speedtest-go/blob/master/LICENSE)
