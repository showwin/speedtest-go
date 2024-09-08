# SpeedTest Specifications
This document records some of the interfaces defined in speedtest for reference only.

## Native Socket Interfaces

The protocol uses a plain text data stream and ends each message with '\n'.
And '\n' and the operators are included in the total bytes.

| Method | Protocol | Describe                                          |
|--------|----------|---------------------------------------------------|
| Greet  | TCP      | Say Hello and get the server version information. |  
| PING   | TCP      | Echo with the server.                             |  
| Loss   | TCP+UDP  | Conduct UDP packet loss test.                     | 
| Down   | TCP      | Sending data to the server.                       |  
| Up     | TCP      | Receive data from the server.                     | 

### Great
```shell
Clinet: HI\n
Server: HELLO [Major].[Minor] ([Major].[Minor].[Patch]) [YYYY]-[MM]-[DD].[LTSCCode].[GitHash]\n
```

### PING
```shell
Clinet: PING [Local Timestamp]\n
Server: PONG [Remote Timestamp]\n
```

### Loss
Please see https://github.com/showwin/speedtest-go/issues/169

### Down
```shell
Clinet: DOWNLOAD [Size]\n
Server: DOWNLOAD [Random Data]\n
```

### Up
```shell
Clinet: UPLOAD [Size]\n
Clinet: [Random Data]
Server: OK [Size] [Timestamp]
```

## References
[1] Reverse Engineering the Speedtest.net Protocol, Gökberk Yaltıraklı https://gist.github.com/sdstrowes/411fca9d900a846a704f68547941eb97
