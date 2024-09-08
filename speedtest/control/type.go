package control

import (
	"strings"
)

type Proto int32

const TypeChunkUndefined = 0

// control protocol and test type
const (
	TypeDownload Proto = 1 << iota
	TypeUpload
	TypeHTTP
	TypeTCP
	TypeICMP
)

func ParseProto(str string) Proto {
	str = strings.ToLower(str)
	if str == "icmp" {
		return TypeICMP
	} else if str == "tcp" {
		return TypeTCP
	} else {
		return TypeHTTP
	}
}

func (p Proto) Assert(u32 Proto) bool {
	return p&u32 == u32
}
