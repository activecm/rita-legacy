package data

import (
	"bytes"
	"github.com/globalsign/mgo/bson"
	"strings"
)

//UniqueIPKey binds an IP to an optional Network UUID which may be used to
//disambiguate local IP addresses residing on different physical networks.
//This index type should be used when determining the equality of UniqueIPs.
type UniqueIPKey struct {
	IP          string       `bson:"ip"`
	NetworkUUID *bson.Binary `bson:"network_uuid,omitempty"`
}

//Equal checks if two UniqueIPKeys have the same IP and Network UUID fields
func (u UniqueIPKey) Equal(ip UniqueIPKey) bool {
	return (u.IP == ip.IP && (u.NetworkUUID == ip.NetworkUUID ||
		(u.NetworkUUID != nil && ip.NetworkUUID != nil &&
			u.NetworkUUID.Kind == ip.NetworkUUID.Kind &&
			bytes.Equal(u.NetworkUUID.Data, ip.NetworkUUID.Data))))
}

//MapKey generates a string which may be used to index a given UniqueIPKey. Concatenates IP and Network UUID.
func (u UniqueIPKey) MapKey() string {
	var builder strings.Builder
	uuidLen := 0
	if u.NetworkUUID != nil {
		uuidLen = 1 + len(u.NetworkUUID.Data)
	}

	builder.Grow(len(u.IP) + uuidLen)
	builder.WriteString(u.IP)
	if u.NetworkUUID != nil {
		builder.WriteByte(u.NetworkUUID.Kind)
		builder.Write(u.NetworkUUID.Data)
	}

	return builder.String()
}

//BSONKey generates a BSON map which may be used to index a given UniqueIPKey. Includes IP and Network UUID.
func (u UniqueIPKey) BSONKey() bson.M {
	key := bson.M{
		"ip": u.IP,
	}
	if u.NetworkUUID != nil {
		key["network_uuid"] = u.NetworkUUID
	}
	return key
}

//SrcDstMapKey generates a string which may be used to index an ordered pair of UniqueIPKeys. Concatenates IPs and UUIDs.
func (u UniqueIPKey) SrcDstMapKey(dst UniqueIPKey) string {
	var builder strings.Builder

	srcUUIDLen := 0
	if u.NetworkUUID != nil {
		srcUUIDLen = 1 + len(u.NetworkUUID.Data)
	}

	dstUUIDLen := 0
	if dst.NetworkUUID != nil {
		dstUUIDLen = 1 + len(dst.NetworkUUID.Data)
	}

	builder.Grow(len(u.IP) + srcUUIDLen + len(dst.IP) + dstUUIDLen)
	builder.WriteString(u.IP)
	builder.WriteString(dst.IP)
	if u.NetworkUUID != nil {
		builder.WriteByte(u.NetworkUUID.Kind)
		builder.Write(u.NetworkUUID.Data)
	}
	if dst.NetworkUUID != nil {
		builder.WriteByte(dst.NetworkUUID.Kind)
		builder.Write(dst.NetworkUUID.Data)
	}
	return builder.String()
}

//SrcDstBSONKey generates a BSON map which may be used to index a given source/destination UniqueIPKey pair.
//Includes IP and Network UUID.
func (u UniqueIPKey) SrcDstBSONKey(dst UniqueIPKey) bson.M {
	key := bson.M{
		"src": u.IP,
		"dst": dst.IP,
	}
	if u.NetworkUUID != nil {
		key["src_network_uuid"] = u.NetworkUUID
	}
	if dst.NetworkUUID != nil {
		key["dst_network_uuid"] = dst.NetworkUUID
	}
	return key
}

//UniqueIP binds an IP to an optional Network UUID and Network Name.
//The UUID and Name serve to diffferentiate local IP addresses
//appearing on distinct physical networks. The Network Name should
//not be considered when determining equality. Use the UniqueIPKey
//sub-type instead.
type UniqueIP struct {
	UniqueIPKey `bson:",inline"`
	NetworkName *string `bson:"network_name,omitempty"`
}

//Equal checks if two UniqueIPs have the same UniqueIPKeys
func (u UniqueIP) Equal(ip UniqueIP) bool {
	return u.UniqueIPKey.Equal(ip.UniqueIPKey)
}

//SrcDstMapKey generates a string which may be used to index an ordered pair of UniqueIPs. Concatenates IPs and UUIDs.
func (u UniqueIP) SrcDstMapKey(dst UniqueIP) string {
	return u.UniqueIPKey.SrcDstMapKey(dst.UniqueIPKey)
}

//SrcDstBSONKey generates a BSON map which may be used to index a given source/destination UniqueIP pair.
//Includes IP and Network UUID.
func (u UniqueIP) SrcDstBSONKey(dst UniqueIP) bson.M {
	return u.UniqueIPKey.SrcDstBSONKey(dst.UniqueIPKey)
}

//UniqueIPSet is a set of UniqueIPs which contains at most one instance of each UniqueIP
//this implementation is based on a slice of UniqueIPs rather than a map[string]UniqueIP
//since it requires less RAM.
type UniqueIPSet []UniqueIP

//Insert adds a UniqueIP to the set
func (s *UniqueIPSet) Insert(ip UniqueIP) {
	contained := s.Contains(ip)
	if contained {
		return
	}
	*s = append(*s, ip)
}

//Contains checks if a given UniqueIP is in the set
func (s UniqueIPSet) Contains(ip UniqueIP) bool {
	for i := range s {
		if s[i].Equal(ip) {
			return true
		}
	}
	return false
}
