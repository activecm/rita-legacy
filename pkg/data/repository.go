package data

import (
	"github.com/globalsign/mgo/bson"
	"strings"
)

type UniqueIP struct {
	IP          string
	NetworkUUID *bson.Binary
	NetworkName *string
}

func (u UniqueIP) BSONQuery(ipField, networkUUIDField string) bson.M {
	return bson.M{
		ipField:          u.IP,
		networkUUIDField: u.NetworkUUID,
	}
}

func (u UniqueIP) SrcDstBSONQuery(dst UniqueIP, srcIPField, dstIPField, srcNetworkUUIDField, dstNetworkUUIDField string) bson.M {
	return bson.M{
		srcIPField:          u.IP,
		dstIPField:          dst.IP,
		srcNetworkUUIDField: u.NetworkUUID,
		dstNetworkUUIDField: dst.NetworkUUID,
	}
}

//MapKey generates a string which may be used to index a given UniqueIP. Concatenates IP and UUID.
func (u UniqueIP) MapKey() string {
	var builder strings.Builder
	uuidLen := 0
	if u.NetworkUUID != nil {
		uuidLen = len(u.NetworkUUID.Data)
	}

	builder.Grow(len(u.IP) + 1 + uuidLen)
	builder.WriteString(u.IP)
	builder.WriteByte(0xFF)
	if u.NetworkUUID != nil {
		builder.Write(u.NetworkUUID.Data)
	}

	return builder.String()
}

//SrcDstMapKey generates a string which may be used to index an ordered pair of UniqueIPs. Concatenates IPs and UUIDs.
func (u UniqueIP) SrcDstMapKey(dst UniqueIP) string {
	var builder strings.Builder

	srcUUIDLen := 0
	if u.NetworkUUID != nil {
		srcUUIDLen = len(u.NetworkUUID.Data)
	}

	dstUUIDLen := 0
	if dst.NetworkUUID != nil {
		dstUUIDLen = len(dst.NetworkUUID.Data)
	}

	builder.Grow(len(u.IP) + 1 + srcUUIDLen + 1 + len(dst.IP) + 1 + dstUUIDLen)
	builder.WriteString(u.IP)
	builder.WriteByte(0xFF)
	if u.NetworkUUID != nil {
		builder.Write(u.NetworkUUID.Data)
	}
	builder.WriteByte(0xFF)
	builder.WriteString(dst.IP)
	builder.WriteByte(0xFF)
	if dst.NetworkUUID != nil {
		builder.Write(dst.NetworkUUID.Data)
	}
	return builder.String()
}
