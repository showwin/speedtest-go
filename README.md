# speedtest-go
Command Line Interface to Test Internet Speed using [speedtest.net](http://www.speedtest.net/)

Inspired by [sivel/speedtest-cli](https://github.com/sivel/speedtest-cli)

## Installation
### OS X (homebrew)
```
$ brew tap showwin/speedtest
$ brew install speedtest

### How to Update ###
$ brew update
$ brew upgrate speedtest
```

## Usage
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

**Select Closest Server by Default**
![](https://github.com/showwin/speedtest-go/blob/master/docs/images/usage_closest.png)

**Select Multiple Server with Server IDs**
![](https://github.com/showwin/speedtest-go/blob/master/docs/images/usage_multi.png)

##LICENSE
[MIT](https://github.com/showwin/speedtest-go/blob/master/LICENSE)
