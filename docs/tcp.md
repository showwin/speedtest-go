# TCP Speed Test

speedtest-go supports the plaintext TCP protocol used by speedtest.net test servers.
HTTP remains the default, so existing commands do not change behavior.

## Usage

Run a TCP download and upload test with:

```bash
speedtest --protocol=tcp
```

The server must expose its speedtest TCP service, normally on port `8080`. The
TCP endpoint is taken from the server `Host` field. HTTP proxies are not used
for TCP connections.

`--ping-mode` is independent from `--protocol`: it controls latency measurement,
while `--protocol` controls download and upload traffic.

## Wire protocol

Each test connection starts with a handshake:

```text
HI
HELLO <server version>
```

Download requests have the following form:

```text
DOWNLOAD <bytes>\n
```

The server then returns exactly `<bytes>` bytes. The client streams these bytes
directly into the existing speed and byte counters.

Upload requests have the following form:

```text
UPLOAD <total bytes> 0\n
<payload>\n
```

The declared total includes the command line, its newline, the payload, and the
payload's final newline. The server confirms the request with:

```text
OK <bytes> <timestamp>
```

The acknowledged byte count is used for the upload confirmation ratio.

## Packet loss

The packet-loss analyzer is independent of the download/upload protocol. TCP
mode still runs the existing TCP-control plus UDP-data packet-loss analysis.
