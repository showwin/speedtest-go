# speedtest-fpngfw
Command Line Interface to Test Internet Speed using [speedtest.net](http://www.speedtest.net/) from Forcepoint Next Gen Firewalls.

Inspired by [showwin/speedtest-go](https://github.com/showwin/speedtest-go)

## Installation

```
wget -q --no-check-certificate https://github.com/Newlode/speedtest-fpngfw/releases/download/v1.0.7/speedtest-fpngfw_1.0.7_Linux_x86_64.tar.gz  -O - | tar -xz speedtest-fpngfw
```

## Usage
```
$ speedtest-fpngfw --help
usage: speedtest-fpngfw [<flags>]

Run a speedtest from a Forcepoint NGFW.

Flags:
      --help               Show context-sensitive help (also try --help-long and --help-man).
  -i, --insecure           Disable TLS certificate verify
  -I, --iface=IFACE        Force the use of IFACE for this test
  -l, --list               Show available speedtest.net servers
  -s, --server=SERVER ...  Select server id to speedtest
  -t, --timeout=TIMEOUT    Define timeout seconds. Default: 10 sec
```

### Choose a specific net-link

You can specify an ethernet interface from which to test.
```
./speedtest-fpngfw -I eth0_3
Source : 192.168.0.253/80.X.X.165 (Orange) [48.7582,2.3987]
Target : 23282 307.56km (Rennes/France) by ORANGE FRANCE

Latency       : 6.215ms
Download Test : ................
Upload Test   : ................
 
Download      : 290.48 Mbit/s
Upload        : 232.43 Mbit/s
```


## LICENSE

[MIT](https://github.com/showwin/speedtest-go/blob/master/LICENSE)
