package data

import (
	"github.com/globalsign/mgo/bson"
	"strings"
)

type UniqueIP struct {
	IP          string      `bson:"ip"`
	NetworkUUID bson.Binary `bson:"network_uuid,omitempty"`
	NetworkName string      `bson:"network_name,omitempty"`
}

//MapKey generates a string which may be used to index a given UniqueIP. Concatenates IP and UUID.
func (u UniqueIP) MapKey() string {
	var builder strings.Builder
	builder.Grow(len(u.IP) + 1 + len(u.NetworkUUID.Data))
	builder.WriteString(u.IP)
	builder.WriteByte(0xFF)
	builder.Write(u.NetworkUUID.Data)
	return builder.String()
}

//SrcDstMapKey generates a string which may be used to index an ordered pair of UniqueIPs. Concatenates IPs and UUIDs.
func (u UniqueIP) SrcDstMapKey(dst UniqueIP) string {
	var builder strings.Builder
	builder.Grow(len(u.IP) + 1 + len(u.NetworkUUID.Data) + 1 + len(dst.IP) + 1 + len(dst.NetworkUUID.Data))
	builder.WriteString(u.IP)
	builder.WriteByte(0xFF)
	builder.Write(u.NetworkUUID.Data)
	builder.WriteByte(0xFF)
	builder.WriteString(dst.IP)
	builder.WriteByte(0xFF)
	builder.Write(dst.NetworkUUID.Data)
	return builder.String()
}
