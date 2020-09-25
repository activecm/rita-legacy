package data

import (
	"bytes"
	"github.com/globalsign/mgo/bson"
	"strings"
)

//UniqueIP binds an IP to an optional Network UUID and Network Name.
//The UUID and Name serve to diffferentiate local IP addresses
//appearing on distinct physical networks. The Network Name should
//not be considered when determining equality.
type UniqueIP struct {
	IP          string       `bson:"ip"`
	NetworkUUID *bson.Binary `bson:"network_uuid,omitempty"`
	NetworkName *string      `bson:"network_name,omitempty"`
}

//Equal checks if two UniqueIPs have the same UniqueIPKeys
func (u UniqueIP) Equal(ip UniqueIP) bool {
	return (u.IP == ip.IP && (u.NetworkUUID == ip.NetworkUUID ||
		(u.NetworkUUID != nil && ip.NetworkUUID != nil &&
			u.NetworkUUID.Kind == ip.NetworkUUID.Kind &&
			bytes.Equal(u.NetworkUUID.Data, ip.NetworkUUID.Data))))
}

//MapKey generates a string which may be used to index a given UniqueIP. Concatenates IP and Network UUID.
func (u UniqueIP) MapKey() string {
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

//BSONKey generates a BSON map which may be used to index a given UniqueIP. Includes IP and Network UUID.
func (u UniqueIP) BSONKey() bson.M {
	key := bson.M{
		"ip": u.IP,
	}
	if u.NetworkUUID != nil {
		key["network_uuid"] = u.NetworkUUID
	}
	return key
}

//UniqueIPPair binds a pair of UniqueIPs where direction matters.
type UniqueIPPair struct {
	SrcIP          string       `bson:"src"`
	SrcNetworkUUID *bson.Binary `bson:"src_network_uuid,omitempty"`
	SrcNetworkName *string      `bson:"src_network_name,omitempty"`
	DstIP          string       `bson:"dst"`
	DstNetworkUUID *bson.Binary `bson:"dst_network_uuid,omitempty"`
	DstNetworkName *string      `bson:"dst_network_name,omitempty"`
}

//NewUniqueIPPair binds a pair of UniqueIPs where direction matters.
func NewUniqueIPPair(source UniqueIP, destination UniqueIP) UniqueIPPair {
	return UniqueIPPair{
		SrcIP:          source.IP,
		DstIP:          destination.IP,
		SrcNetworkUUID: source.NetworkUUID,
		DstNetworkUUID: destination.NetworkUUID,
		SrcNetworkName: source.NetworkName,
		DstNetworkName: destination.NetworkName,
	}
}

//Source returns the source UniqueIP from the pair.
func (p UniqueIPPair) Source() UniqueIP {
	return UniqueIP{
		IP:          p.SrcIP,
		NetworkUUID: p.SrcNetworkUUID,
		NetworkName: p.SrcNetworkName,
	}
}

//Destination returns the destination UniqueIP from the pair.
func (p UniqueIPPair) Destination() UniqueIP {
	return UniqueIP{
		IP:          p.DstIP,
		NetworkUUID: p.DstNetworkUUID,
		NetworkName: p.DstNetworkName,
	}
}

//MapKey generates a string which may be used to index an ordered pair of UniqueIPs. Concatenates IPs and UUIDs.
func (p UniqueIPPair) MapKey() string {
	var builder strings.Builder

	srcUUIDLen := 0
	if p.SrcNetworkUUID != nil {
		srcUUIDLen = 1 + len(p.SrcNetworkUUID.Data)
	}

	dstUUIDLen := 0
	if p.DstNetworkUUID != nil {
		dstUUIDLen = 1 + len(p.DstNetworkUUID.Data)
	}

	builder.Grow(len(p.SrcIP) + srcUUIDLen + len(p.DstIP) + dstUUIDLen)
	builder.WriteString(p.SrcIP)
	builder.WriteString(p.DstIP)
	if p.SrcNetworkUUID != nil {
		builder.WriteByte(p.SrcNetworkUUID.Kind)
		builder.Write(p.SrcNetworkUUID.Data)
	}
	if p.DstNetworkUUID != nil {
		builder.WriteByte(p.DstNetworkUUID.Kind)
		builder.Write(p.DstNetworkUUID.Data)
	}
	return builder.String()
}

//BSONKey generates a BSON map which may be used to index a given source/destination UniqueIP pair.
//Includes IP and Network UUID.
func (p UniqueIPPair) BSONKey() bson.M {
	key := bson.M{
		"src": p.SrcIP,
		"dst": p.DstIP,
	}
	if p.SrcNetworkUUID != nil {
		key["src_network_uuid"] = p.SrcNetworkUUID
	}
	if p.DstNetworkUUID != nil {
		key["dst_network_uuid"] = p.DstNetworkUUID
	}
	return key
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
