package data

import (
	"strings"

	"github.com/globalsign/mgo/bson"
)

//UniqueSrcFQDNPair is used to make a tuple of
// Src IP/UUID/Name and an FQDN to which the Src IP
// was attempting to communicate
type UniqueSrcFQDNPair struct {
	UniqueSrcIP `bson:",inline"`
	FQDN        string `bson:"fqdn"`
}

//NewUniqueSrcFQDNPair binds a pair of UniqueIPs and an FQDN
func NewUniqueSrcFQDNPair(source UniqueIP, fqdn string) UniqueSrcFQDNPair {
	return UniqueSrcFQDNPair{
		UniqueSrcIP: UniqueSrcIP{
			SrcIP:          source.IP,
			SrcNetworkUUID: source.NetworkUUID,
			SrcNetworkName: source.NetworkName,
		},
		FQDN: fqdn,
	}
}

//MapKey generates a string which may be used to index a Unique SrcIP / FQDN pair. Concatenates IPs and UUIDs.
func (p UniqueSrcFQDNPair) MapKey() string {
	var builder strings.Builder

	srcUUIDLen := 1 + len(p.SrcNetworkUUID.Data)

	builder.Grow(len(p.SrcIP) + srcUUIDLen + len(p.FQDN))
	builder.WriteString(p.SrcIP)
	builder.WriteByte(p.SrcNetworkUUID.Kind)
	builder.Write(p.SrcNetworkUUID.Data)

	builder.WriteString(p.FQDN)

	return builder.String()
}

//BSONKey generates a BSON map which may be used to index a given a unique
// src-fqdn pair. Includes IP and Network UUID.
func (p UniqueSrcFQDNPair) BSONKey() bson.M {
	key := bson.M{
		"src":              p.SrcIP,
		"src_network_uuid": p.SrcNetworkUUID,
		"fqdn":             p.FQDN,
	}
	return key
}
