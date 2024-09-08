package transport

import (
	"bytes"
	"context"
	"fmt"
	mrand "math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

var (
	loss = []byte{0x4c, 0x4f, 0x53, 0x53}
)

type PacketLossSender struct {
	ID            string   // UUID
	nounce        int64    // Random int (maybe) [0,10000000000)
	withTimestamp bool     // With timestamp (ten seconds level)
	conn          net.Conn // UDP Conn
	raw           []byte
	host          string
	dialer        *net.Dialer
}

func NewPacketLossSender(uuid string, dialer *net.Dialer) (*PacketLossSender, error) {
	rd := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	nounce := rd.Int63n(10000000000)
	p := &PacketLossSender{
		ID:            strings.ToUpper(uuid),
		nounce:        nounce,
		withTimestamp: false, // we close it as default, we won't be able to use it right now.
		dialer:        dialer,
	}
	p.raw = []byte(fmt.Sprintf("%s %d %s %s", loss, nounce, "#", uuid))
	return p, nil
}

func (ps *PacketLossSender) Connect(ctx context.Context, host string) (err error) {
	ps.host = host
	ps.conn, err = ps.dialer.DialContext(ctx, "udp", ps.host)
	return err
}

// Send
// @param order the value will be sent
func (ps *PacketLossSender) Send(order int) error {
	payload := bytes.Replace(ps.raw, []byte{0x23}, []byte(strconv.Itoa(order)), 1)
	_, err := ps.conn.Write(payload)
	return err
}
