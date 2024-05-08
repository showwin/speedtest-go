# speedtest-go
**Command Line Interface and pure [Go API](#go-api) to Test Internet Speed using [speedtest.net](http://www.speedtest.net/)**.

You can speedtest 2x faster than [speedtest.net](http://www.speedtest.net/) with almost the same result. [See the experimental results](https://github.com/showwin/speedtest-go#summary-of-experimental-results).
Inspired by [sivel/speedtest-cli](https://github.com/sivel/speedtest-cli)

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

#### [Nix](https://nixos.org) (package manager)
```bash
# Enter the latest speedtest-go environment
$ nix-shell -p speedtest-go
```

#### Other Platforms (Linux, Windows, etc.)

Please download the compatible package from [Releases](https://github.com/showwin/speedtest-go/releases).
If there are no compatible packages you want, please let me know on [Issue Tracker](https://github.com/showwin/speedtest-go/issues).

### Usage

```bash
$ speedtest --help
usage: speedtest-go [<flags>]

Flags:
      --help                   Show context-sensitive help (also try --help-long and --help-man).
  -l, --list                   Show available speedtest.net servers.
  -s, --server=SERVER ...      Select server id to speedtest.
      --custom-url=CUSTOM-URL  Specify the url of the server instead of fetching from speedtest.net.
      --saving-mode            Test with few resources, though low accuracy (especially > 30Mbps).
      --json                   Output results in json format.
      --location=LOCATION      Change the location with a precise coordinate (format: lat,lon).
      --city=CITY              Change the location with a predefined city label.
      --city-list              List all predefined city labels.
      --proxy=PROXY            Set a proxy(http[s] or socks) for the speedtest.
                               eg: --proxy=socks://10.20.0.101:7890
                               eg: --proxy=http://10.20.0.101:7890
      --source=SOURCE          Bind a source interface for the speedtest.
      --dns-bind-source        DNS request binding source (experimental).
                               eg: --source=10.20.0.101
  -m  --multi                  Enable multi-server mode.
  -t  --thread=THREAD          Set the number of concurrent connections.
      --search=SEARCH          Fuzzy search servers by a keyword.
      --ua                     Set the user-agent header for the speedtest.
      --no-download            Disable download test.
      --no-upload              Disable upload test.
      --ping-mode              Select a method for Ping (support icmp/tcp/http).
  -u  --unit                   Set human-readable and auto-scaled rate units for output 
                               (options: decimal-bits/decimal-bytes/binary-bits/binary-bytes).
  -d  --debug                  Enable debug mode.
      --version                Show application version.
```

#### Test Internet Speed

Simply use `speedtest` command. The closest server is selected by default. Use the `-m` flag to enable multi-measurement mode (recommended)

```bash
$ speedtest

    speedtest-go v1.7.2 @showwin

✓ ISP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]
✓ Found 20 Public Servers

✓ Test Server: [6691] 9.03km Shizuoka (Japan) by sudosan
✓ Latency: 4.452963ms Jitter: 41.271µs Min: 4.395179ms Max: 4.517576ms
✓ Download: 115.52 Mbps (Used: 135.75MB) (Latency: 4ms Jitter: 0ms Min: 4ms Max: 4ms)
✓ Upload: 4.02 Mbps (Used: 6.85MB) (Latency: 4ms Jitter: 1ms Min: 3ms Max: 8ms)
  Packet Loss: 3.36%
```

#### Test with Other Servers

If you want to select other servers to test, you can see the available server list.

```bash
$ speedtest --list
Testing From IP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]
[6691]     9.03km   32.3365ms  Shizuoka (Japan) by sudosan
[6087]   120.55km   51.7453ms  Fussa-shi (Japan) by Allied Telesis Capital Corporation
[6508]   125.44km   54.6683ms  Yokohama (Japan) by at2wn
[6424]   148.23km   61.4724ms  Tokyo (Japan) by Cordeos Corp.
...
```

and select them by id.

```bash
$ speedtest --server 6691 --server 6087

    speedtest-go v1.7.2 @showwin

✓ ISP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]
✓ Found 2 Specified Public Server(s)

✓ Test Server: [6691] 9.03km Shizuoka (Japan) by sudosan
✓ Latency: 21.424ms Jitter: 1.644ms Min: 19.142ms Max: 23.926ms
✓ Download: 65.82Mbps (Used: 75.48MB) (Latency: 22ms Jitter: 2ms Min: 17ms Max: 24ms)
✓ Upload: 27.00Mbps (Used: 36.33MB) (Latency: 23ms Jitter: 2ms Min: 18ms Max: 25ms)
  Packet Loss: 7.55%

✓ Test Server: [6087] 120.55km Fussa-shi (Japan) by Allied Telesis Capital Corporation
✓ Latency: 38.694699ms Jitter: 2.724ms Min: 36.443ms Max: 39.953ms
✓ Download: 72.24Mbps (Used: 83.72MB) (Latency: 37ms Jitter: 3ms Min: 36ms Max: 40ms)
✓ Upload: 29.56Mbps (Used: 47.64MB) (Latency: 38ms Jitter: 3ms Min: 37ms Max: 41ms)
  Packet Loss: 4.33%
```

#### Test with a virtual location

With `--city` or `--location` option, the closest servers of the location will be picked.
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

With `--saving-mode` option, it can be executed even in an insufficient memory environment like IoT devices.
The memory usage can be reduced to 1/10, about 10MB of memory is used.

However, please be careful that the accuracy is particularly low, especially in an environment of 30 Mbps or higher.
To get more accurate results, run multiple times and average.

For more details, please see [saving mode experimental result](https://github.com/showwin/speedtest-go/blob/master/docs/saving_mode_experimental_result.md).

⚠️This feature has been deprecated > v1.4.0, because speedtest-go can always run with less than 10MBytes of memory now. Even so, `--saving-mode` is still a good way to reduce computation.

## Go API

```bash
go get github.com/showwin/speedtest-go
```

### API Usage

The [code](https://github.com/showwin/speedtest-go/blob/master/example/naive/main.go) below finds the closest available speedtest server and tests the latency, download, and upload speeds.
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
	
	// Get user's network information
	// user, _ := speedtestClient.FetchUserInfo()
	
	// Get a list of servers near a specified location
	// user.SetLocationByCity("Tokyo")
	// user.SetLocation("Osaka", 34.6952, 135.5006)
    
	// Search server using serverID.
	// eg: fetch server with ID 28910.
	// speedtest.ErrServerNotFound will be returned if the server cannot be found.
	// server, err := speedtest.FetchServerByID("28910")
	
	serverList, _ := speedtestClient.FetchServers()
	targets, _ := serverList.FindServer([]int{})

	for _, s := range targets {
		// Please make sure your host can access this test server,
		// otherwise you will get an error.
		// It is recommended to replace a server at this time
		s.PingTest(nil)
		s.DownloadTest()
		s.UploadTest()
		// Note: The unit of s.DLSpeed, s.ULSpeed is bytes per second, this is a float64.
		fmt.Printf("Latency: %s, Download: %s, Upload: %s\n", s.Latency, s.DLSpeed, s.ULSpeed)
		s.Context.Reset() // reset counter
	}
}
```

The [code](https://github.com/showwin/speedtest-go/blob/master/example/packet_loss/main.go) will find the closest available speedtest server and analyze packet loss.
```go
package main

import (
	"fmt"
	"github.com/showwin/speedtest-go/speedtest"
	"github.com/showwin/speedtest-go/speedtest/transport"
)

// Note: The current packet loss analyzer does not support udp over http.
// This means we cannot get packet loss through a proxy.
func main() {
	// Retrieve available servers
	var speedtestClient = speedtest.New()
	serverList, _ := speedtestClient.FetchServers()
	targets, _ := serverList.FindServer([]int{})

	// Create a packet loss analyzer, use default options
	analyzer, err := speedtest.NewPacketLossAnalyzer(nil)
	checkError(err)

	// Perform packet loss analysis on all available servers
	for _, server := range targets {
		err = analyzer.Run(server.Host, func(packetLoss *transport.PLoss) {
			fmt.Println(packetLoss, server.Host, server.Name)
			// fmt.Println(packetLoss.Loss())
		})
		checkError(err)
	}
}
```

## Summary of Experimental Results

Speedtest-go is a great tool because of the following 4 reasons:
* Cross-platform available.
* Low memory environment.
* We are the first open source project to implement all features base on speedtest.net, including down/up rates, jitter and packet loss, etc.
* Testing time is the **SHORTEST** compare to [speedtest.net](http://www.speedtest.net/) and [sivel/speedtest-cli](https://github.com/sivel/speedtest-cli), especially about 2x faster than [speedtest.net](http://www.speedtest.net/).
* Result is **MORE CLOSE** to [speedtest.net](http://www.speedtest.net/) than [speedtest-cli](https://github.com/sivel/speedtest-cli).

The following data is summarized. If you got interested, please see [more details](https://github.com/showwin/speedtest-go/blob/master/docs/experimental_result.md).

### Download (Mbps)

distance = distance to testing server
* 0 - 1000(km) ≒ domestic
* 1000 - 8000(km) ≒ same region
* 8000 - 20000(km) ≒ really far!
* 20000km is half of the circumference of our planet.

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

See [Contributors](https://github.com/showwin/speedtest-go/graphs/contributors), PRs are welcome!

## Issues

You can find or report issues in the [Issue Tracker](https://github.com/showwin/speedtest-go/issues).

## LICENSE

[MIT](https://github.com/showwin/speedtest-go/blob/master/LICENSE)
