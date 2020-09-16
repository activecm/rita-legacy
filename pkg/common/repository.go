package common

import (
	"github.com/globalsign/mgo/bson"
	"strings"
)

type UniqueIP struct {
	IP          string      `bson:"ip"`
	NetworkUUID bson.Binary `bson:"network_uuid,omitempty"`
	NetworkName string      `bson:"network_name,omitempty"`
}

//Create a new UniqueIP for a publicly routable IP. Sets UniqueIP.NetworkUUID and UniqueIP.NetworkName to zero values.
func NewPublicIP(ip string) UniqueIP {
	return UniqueIP{
		IP: ip,
	}
}

//Create a ne UniqueIP for a private IP. Requires a network UUID and name to disambiguate separate local networks.
func NewLocalIP(ip string, networkUUID [16]byte, networkName string) UniqueIP {
	return UniqueIP{
		IP: ip,
		NetworkUUID: bson.Binary{
			Kind: bson.BinaryUUID,
			Data: networkUUID[:],
		},
		NetworkName: networkName,
	}
}

//UniqueIPMapKey generates a string which may be used to index a given UniqueIP. Concatenates IP and UUID.
func UniqueIPMapKey(u UniqueIP) string {
	var builder strings.Builder
	builder.Grow(len(u.IP) + 1 + len(u.NetworkUUID.Data))
	builder.WriteString(u.IP)
	builder.WriteByte(0xFF)
	builder.Write(u.NetworkUUID.Data)
	return builder.String()
}

//UniqueSrcDstIPMapKey generates a string which may be used to index an ordered pair of UniqueIPs. Concatenates IPs and UUIDs.
func UniqueSrcDstIPMapKey(src UniqueIP, dst UniqueIP) string {
	var builder strings.Builder
	builder.Grow(len(src.IP) + 1 + len(src.NetworkUUID.Data) + 1 + len(dst.IP) + 1 + len(dst.NetworkUUID.Data))
	builder.WriteString(src.IP)
	builder.WriteByte(0xFF)
	builder.Write(src.NetworkUUID.Data)
	builder.WriteByte(0xFF)
	builder.WriteString(dst.IP)
	builder.WriteByte(0xFF)
	builder.Write(dst.NetworkUUID.Data)
	return builder.String()
}
