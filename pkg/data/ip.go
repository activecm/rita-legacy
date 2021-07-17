package data

import (
	"bytes"
	"net"
	"strings"

	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

//UniqueIP binds an IP to an optional Network UUID and Network Name.
//The UUID and Name serve to diffferentiate local IP addresses
//appearing on distinct physical networks. The Network Name should
//not be considered when determining equality.
type UniqueIP struct {
	IP          string      `bson:"ip"`
	NetworkUUID bson.Binary `bson:"network_uuid"`
	NetworkName string      `bson:"network_name"`
}

//NewUniqueIP returns a new UniqueIP. If the given ip is publicly routable, the resulting UniqueIP's
//NetworkUUID and NetworkName will be set to PublicNetworkUUID and PublicNetworkName respectively.
//Otherwise, the NetworkUUID and NetworkName will be set based on the provided agentName and agentUUID.
//If the provided agent data is invalid, the NetworkUUID and NetworkName will be set to
//UnknownPrivateNetworkUUID and UnknownPrivateNetworkName.
func NewUniqueIP(ip net.IP, agentUUID, agentName string) UniqueIP {
	u := UniqueIP{}
	u.IP = ip.String()

	if util.IPIsPubliclyRoutable(ip) {
		u.NetworkName = util.PublicNetworkName
		u.NetworkUUID = util.PublicNetworkUUID
		return u
	}

	// agent information is optional, provide a fast path to avoid calling uuid.Parse with invalid data
	if len(agentUUID) == 0 || len(agentName) == 0 {
		u.NetworkName = util.UnknownPrivateNetworkName
		u.NetworkUUID = util.UnknownPrivateNetworkUUID
		return u
	}

	id, err := uuid.Parse(agentUUID)
	if err != nil {
		u.NetworkName = util.UnknownPrivateNetworkName
		u.NetworkUUID = util.UnknownPrivateNetworkUUID
		return u
	}

	u.NetworkUUID = bson.Binary{
		Kind: bson.BinaryUUID,
		Data: id[:],
	}
	u.NetworkName = agentName
	return u
}

//Equal checks if two UniqueIPs have the same IP and network UUID
func (u UniqueIP) Equal(ip UniqueIP) bool {
	return (u.IP == ip.IP &&
		u.NetworkUUID.Kind == ip.NetworkUUID.Kind &&
		bytes.Equal(u.NetworkUUID.Data, ip.NetworkUUID.Data))
}

//MapKey generates a string which may be used to index a given UniqueIP. Concatenates IP and Network UUID.
func (u UniqueIP) MapKey() string {
	var builder strings.Builder
	builder.Grow(len(u.IP) + 1 + len(u.NetworkUUID.Data))
	builder.WriteString(u.IP)
	builder.WriteByte(u.NetworkUUID.Kind)
	builder.Write(u.NetworkUUID.Data)

	return builder.String()
}

//BSONKey generates a BSON map which may be used to index a given UniqueIP. Includes IP and Network UUID.
func (u UniqueIP) BSONKey() bson.M {
	key := bson.M{
		"ip":           u.IP,
		"network_uuid": u.NetworkUUID,
	}
	return key
}

//UniqueSrcIP is a unique IP which acts as the source in an IP pair
type UniqueSrcIP struct {
	SrcIP          string      `bson:"src"`
	SrcNetworkUUID bson.Binary `bson:"src_network_uuid"`
	SrcNetworkName string      `bson:"src_network_name"`
}

//AsSrc returns the UniqueIP in the UniqueSrcIP format
func (u UniqueIP) AsSrc() UniqueSrcIP {
	return UniqueSrcIP{
		SrcIP:          u.IP,
		SrcNetworkUUID: u.NetworkUUID,
		SrcNetworkName: u.NetworkName,
	}
}

//Unpair returns a copy of the SrcUniqueIP in UniqueIP format
func (u UniqueSrcIP) Unpair() UniqueIP {
	return UniqueIP{
		IP:          u.SrcIP,
		NetworkUUID: u.SrcNetworkUUID,
		NetworkName: u.SrcNetworkName,
	}
}

//BSONKey generates a BSON map which may be used to index a the source of a UniqueIP pair.
//Includes IP and Network UUID.
func (u UniqueSrcIP) BSONKey() bson.M {
	key := bson.M{
		"src":              u.SrcIP,
		"src_network_uuid": u.SrcNetworkUUID,
	}
	return key
}

//UniqueDstIP is a unique IP which acts as the destination in an IP Pair
type UniqueDstIP struct {
	DstIP          string      `bson:"dst"`
	DstNetworkUUID bson.Binary `bson:"dst_network_uuid"`
	DstNetworkName string      `bson:"dst_network_name"`
}

//AsDst returns the UniqueIP in the UniqueDstIP format
func (u UniqueIP) AsDst() UniqueDstIP {
	return UniqueDstIP{
		DstIP:          u.IP,
		DstNetworkUUID: u.NetworkUUID,
		DstNetworkName: u.NetworkName,
	}
}

//Unpair returns a copy of the DstUniqueIP in UniqueIP format
func (u UniqueDstIP) Unpair() UniqueIP {
	return UniqueIP{
		IP:          u.DstIP,
		NetworkUUID: u.DstNetworkUUID,
		NetworkName: u.DstNetworkName,
	}
}

//BSONKey generates a BSON map which may be used to index a the destination of a UniqueIP pair.
//Includes IP and Network UUID.
func (u UniqueDstIP) BSONKey() bson.M {
	key := bson.M{
		"dst":              u.DstIP,
		"dst_network_uuid": u.DstNetworkUUID,
	}
	return key
}

//UniqueIPPair binds a pair of UniqueIPs where direction matters.
type UniqueIPPair struct {
	UniqueSrcIP `bson:",inline"`
	UniqueDstIP `bson:",inline"`
}

//NewUniqueIPPair binds a pair of UniqueIPs where direction matters.
func NewUniqueIPPair(source UniqueIP, destination UniqueIP) UniqueIPPair {
	return UniqueIPPair{
		UniqueSrcIP: UniqueSrcIP{
			SrcIP:          source.IP,
			SrcNetworkUUID: source.NetworkUUID,
			SrcNetworkName: source.NetworkName,
		},
		UniqueDstIP: UniqueDstIP{
			DstIP:          destination.IP,
			DstNetworkUUID: destination.NetworkUUID,
			DstNetworkName: destination.NetworkName,
		},
	}
}

//MapKey generates a string which may be used to index an ordered pair of UniqueIPs. Concatenates IPs and UUIDs.
func (p UniqueIPPair) MapKey() string {
	var builder strings.Builder

	srcUUIDLen := 1 + len(p.SrcNetworkUUID.Data)
	dstUUIDLen := 1 + len(p.DstNetworkUUID.Data)

	builder.Grow(len(p.SrcIP) + srcUUIDLen + len(p.DstIP) + dstUUIDLen)
	builder.WriteString(p.SrcIP)
	builder.WriteString(p.DstIP)
	builder.WriteByte(p.SrcNetworkUUID.Kind)
	builder.Write(p.SrcNetworkUUID.Data)
	builder.WriteByte(p.DstNetworkUUID.Kind)
	builder.Write(p.DstNetworkUUID.Data)

	return builder.String()
}

//BSONKey generates a BSON map which may be used to index a given source/destination UniqueIP pair.
//Includes IP and Network UUID.
func (p UniqueIPPair) BSONKey() bson.M {
	key := bson.M{
		"src":              p.SrcIP,
		"src_network_uuid": p.SrcNetworkUUID,
		"dst":              p.DstIP,
		"dst_network_uuid": p.DstNetworkUUID,
	}
	return key
}

//UniqueIPSet is a set of UniqueIPs which contains at most one instance of each UniqueIP
//this implementation is based on a slice of UniqueIPs rather than a map[string]UniqueIP
//since it requires less RAM.
type UniqueIPSet map[string]UniqueIP

//Items returns the UniqueIPs in the set as a slice.
func (s UniqueIPSet) Items() []UniqueIP {
	retVal := make([]UniqueIP, 0, len(s))
	for _, ip := range s {
		retVal = append(retVal, ip)
	}
	return retVal
}

//Insert adds a UniqueIP to the set
func (s UniqueIPSet) Insert(ip UniqueIP) {
	if s == nil {
		s = make(UniqueIPSet)
	}
	s[ip.MapKey()] = ip
}

//Contains checks if a given UniqueIP is in the set
func (s UniqueIPSet) Contains(ip UniqueIP) bool {
	if s == nil {
		return false
	}
	_, ok := s[ip.MapKey()]
	return ok
}
