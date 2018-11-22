# speedtest-go
##### Command Line Interface and pure Go API to Test Internet Speed using [speedtest.net](http://www.speedtest.net/)  
You can speedtest 2x faster than [speedtest.net](http://www.speedtest.net/) with almost the same result. [See the experimental results.](https://github.com/showwin/speedtest-go#summary-of-experimental-results).

This fork allows abstracts the implementation of the speed test to a package so it can be used by other go programs. 

Inspired by [sivel/speedtest-cli](https://github.com/sivel/speedtest-cli)

## API Installation

```
go get github.com/cbergoon/speedtest-go
```

## API Usage
The code below finds closest available speedtest server and tests the latency, download, and upload speeds. 
```go
package main

import (
	"fmt"
	"github.com/cbergoon/speedtest-go/speedtest"
)

func main() {
	user, _ := speedtest.FetchUserInfo()

	serverList, _ := speedtest.FetchServerList(user)
	targets, _ := serverList.FindServer([]int{})

	for _, s := range targets {
		s.PingTest()
		s.DownloadTest()
		s.UploadTest()

		fmt.Printf("Latency: %s, Download: %f, Upload: %f\n", s.Latency, s.DLSpeed, s.ULSpeed)
	}
}
```

## CLI Installation
### OS X (homebrew)
```
$ brew tap showwin/speedtest
$ brew install speedtest

### How to Update ###
$ brew update
$ brew upgrade speedtest
```

### Others (Linux, Windows, etc.)
Please download compatible package from [Releases](https://github.com/showwin/speedtest-go/releases).  
If there are no compatible package you want, please let me know by creating an [issue](https://github.com/showwin/speedtest-go/issues).

## CLI Usage
```
$ speedtest --help
usage: download [<flags>]

Flags:
      --help             Show context-sensitive help (also try --help-long and --help-man).
  -l, --list             Show available speedtest.net servers
  -s, --server=SERVER    Select server id to speedtest
  -t, --timeout=TIMEOUT  Define timeout seconds. Default: 10 sec
      --version          Show application version.
```

#### Select Closest Server by Default
```
$ speedtest
Testing From IP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]

Target Server: [6691]     9.03km Shizuoka (Japan) by sudosan
latency: 39.436061ms
Download Test: ................
Upload Test: ................

Download: 73.30 Mbit/s
Upload: 35.26 Mbit/s
```

#### Show Available Servers
```
$ speedtest --list
Testing From IP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]
[6691]     9.03km Shizuoka (Japan) by sudosan
[6087]   120.55km Fussa-shi (Japan) by Allied Telesis Capital Corporation
[6508]   125.44km Yokohama (Japan) by at2wn
[6424]   148.23km Tokyo (Japan) by Cordeos Corp.
[6492]   153.06km Sumida (Japan) by denpa893
[7139]   192.63km Tsukuba (Japan) by SoftEther Corporation
[6368]   194.83km Maibara (Japan) by gatolabo
[6463]   220.39km Kusatsu (Japan) by j416dy
[6766]   232.54km Nomi (Japan) by JAIST(ino-lab)
[6476]   265.10km Osaka (Japan) by rxy (individual)
[6477]   268.94km Sakai (Japan) by satoweb
...
```

#### Select Multiple Servers by Server ID
```
$ speedtest --server 6691 --server 6087
Testing From IP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]

Target Server: [6691]     9.03km Shizuoka (Japan) by sudosan
Latency: 23.612861ms
Download Test: ................
Upload Test: ........

Target Server: [6087]   120.55km Fussa-shi (Japan) by Allied Telesis Capital Corporation
Latency: 38.694699ms
Download Test: ................
Upload Test: ................

[6691] Download: 65.82 Mbit/s, Upload: 27.00 Mbit/s
[6087] Download: 72.24 Mbit/s, Upload: 29.56 Mbit/s
Download Avg: 69.03 Mbit/s
Upload Avg: 28.28 Mbit/s
```

## Summary of Experimental Results
Speedtest-go is a great tool because of following 2 reasons:
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
| :-- | :--: | :--: | :--: |
| 0 - 1000 | 92.12 | **91.21** | 70.27 |
| 1000 - 8000 | 66.45 | **65.51** | 56.56 |
| 8000 - 20000 | 11.84 | 9.43 | **11.87** |

### Upload (Mbps)


| distance (km) | speedtest.net | speedtest-go | speedtest-cli |
| :-- | :--: | :--: | :--: |
| 0 - 1000 | 65.56 | **47.58** | 36.16 |
| 1000 - 8000 | 58.02 | **54.74** | 26.78 |
| 8000 - 20000 | 5.20 | 8.32 | **2.58** |

### Testing Time (sec)


| distance (km) | speedtest.net | speedtest-go | speedtest-cli |
| :-- | :--: | :--: | :--: |
| 0 - 1000 | 45.03 | **22.84** | 24.46 |
| 1000 - 8000 | 44.89 | **24.45** | 28.52 |
| 8000 - 20000 | 49.64 | **34.08** | 41.26 |

## Contributors
* [kogai](https://github.com/kogai)

## LICENSE

[MIT](https://github.com/showwin/speedtest-go/blob/master/LICENSE)
