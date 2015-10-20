# speedtest-go
Command Line Interface to Test Internet Speed using [speedtest.net](http://www.speedtest.net/)

Inspired by [sivel/speedtest-cli](https://github.com/sivel/speedtest-cli)

## Usage
```
$ go run speedtest.go --help
usage: speedtest [<flags>]

Flags:
      --help           Show context-sensitive help (also try --help-long and --help-man).
  -l, --list           show available speedtest.net servers
  -s, --server=SERVER  select server id to speedtest
```
![](https://github.com/showwin/speedtest-go/blob/master/docs/images/usage.png)

## ToDo
* [x] fetch available servers
* [x] select closest server to test
* [x] measure download speed
* [x] make assets to upload
* [x] measure upload speed
* [ ] better down/upload algorithm for very low bandwidth
* [x] `--server id` option: select server to test
* [ ] `--world` option: measure down/upload speed to world wide servers
* [ ] `--secure` option: use HTTPS instead of HTTP
* [ ] build binary file for measure OS

##LICENSE
[MIT](https://github.com/showwin/speedtest-go/blob/master/LICENSE)
