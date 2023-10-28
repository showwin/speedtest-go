package tcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

var (
	pingPrefix = []byte{0x50, 0x49, 0x4e, 0x47, 0x20}
	// downloadPrefix = []byte{0x44, 0x4F, 0x57, 0x4E, 0x4C, 0x4F, 0x41, 0x44, 0x20}
	// uploadPrefix   = []byte{0x55, 0x50, 0x4C, 0x4F, 0x41, 0x44, 0x20}
	hiFormat   = []byte{0x48, 0x49}
	quitFormat = []byte{0x51, 0x55, 0x49, 0x54}
)

var (
	ErrEchoData  = errors.New("incorrect echo data")
	ErrEmptyConn = errors.New("empty conn")
)

func pingFormat(locTime int64) []byte {
	return strconv.AppendInt(pingPrefix, locTime, 10)
}

type Client struct {
	conn    net.Conn
	host    string
	version string

	dialer *net.Dialer

	reader *bufio.Reader
}

func NewClient(dialer *net.Dialer, host string) *Client {
	return &Client{
		host:   host,
		dialer: dialer,
	}
}

func (client *Client) Connect() (err error) {
	client.conn, err = client.dialer.Dial("tcp", client.host)
	client.reader = bufio.NewReader(client.conn)
	return
}

func (client *Client) Disconnect() (err error) {
	_, _ = client.conn.Write(quitFormat)
	client.conn = nil
	client.reader = nil
	client.version = ""
	return
}

func (client *Client) Write(data []byte) (err error) {
	if client.conn == nil {
		return ErrEmptyConn
	}
	_, err = fmt.Fprintf(client.conn, "%s\n", data)
	return
}

func (client *Client) Read() ([]byte, error) {
	if client.conn == nil {
		return nil, ErrEmptyConn
	}
	return client.reader.ReadBytes('\n')
}

func (client *Client) Version() string {
	if len(client.version) == 0 {
		err := client.Write(hiFormat)
		if err == nil {
			message, err := client.Read()
			if err != nil || len(message) < 8 {
				return "unknown"
			}
			client.version = string(message[6 : len(message)-1])
		}
	}
	return client.version
}

// PingContext Measure latency(RTT) between client and server.
// We use the 2RTT method to obtain three RTT result in
// order to get more data in less time (t2-t0, t4-t2, t3-t1).
// And give lower weight to the delay measured by the server.
// local factor = 0.4 * 2 and remote factor = 0.2
// latency = 0.4 * (t2 - t0) + 0.4 * (t4 - t2) + 0.2 * (t3 - t1)
// @return cumulative delay in nanoseconds
func (client *Client) PingContext(ctx context.Context) (int64, error) {
	resultChan := make(chan error, 1)

	var accumulatedLatency int64 = 0
	var firstReceivedByServer int64 // t1

	go func() {
		for i := 0; i < 2; i++ {
			t0 := time.Now().UnixNano()
			if err := client.Write(pingFormat(t0)); err != nil {
				resultChan <- err
				return
			}
			data, err := client.Read()
			t2 := time.Now().UnixNano()
			if err != nil {
				resultChan <- err
				return
			}
			if len(data) != 19 {
				resultChan <- ErrEchoData
				return
			}
			tx, err := strconv.ParseInt(string(data[5:18]), 10, 64)
			if err != nil {
				resultChan <- err
				return
			}
			accumulatedLatency += (t2 - t0) * 4 / 10 // 0.4
			if i == 0 {
				firstReceivedByServer = tx
			} else {
				// append server-side latency result
				accumulatedLatency += (tx - firstReceivedByServer) * 1000 * 1000 * 2 / 10 // 0.2
			}
		}
		resultChan <- nil
		close(resultChan)
	}()

	select {
	case err := <-resultChan:
		return accumulatedLatency, err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (client *Client) Download() {
	panic("Unimplemented method: Client.Download()")
}

func (client *Client) Upload() {
	panic("Unimplemented method: Client.Upload()")
}
