package uconnproxy

import (
	"strings"

	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

// Repository for uconnproxy collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnProxyMap map[string]*Input)
}

//updateInfo ....
type updateInfo struct {
	selector bson.M
	query    bson.M
}

//update ....
type update struct {
	uconnProxy updateInfo
}

//UniqueSrcHostname is used to make a tuple of
// Src IP/UUID/Name and an FQDN to which the Src IP
// was attempting to communicate
type UniqueSrcHostname struct {
	data.UniqueSrcIP `bson:",inline"`
	FQDN             string `bson:"fqdn"`
}

//Input structure for sending data
//to the analyzer. Contains a tuple of
// Src IP/UUID/Name and an FQDN to which the Src IP
// was attempting to communicate.
// Contains a list of unique time stamps for the
// connections out from the Src to the FQDN via the
// proxy server(s) and a count of the connections.
type Input struct {
	Hosts           UniqueSrcHostname
	TsList          []int64
	ProxyIPs        data.UniqueIPSet
	ConnectionCount int64
}

//NewUniqueSrcHostname binds a pair of UniqueIPs and an FQDN
func NewUniqueSrcHostname(source data.UniqueIP, fqdn string) UniqueSrcHostname {
	return UniqueSrcHostname{
		UniqueSrcIP: data.UniqueSrcIP{
			SrcIP:          source.IP,
			SrcNetworkUUID: source.NetworkUUID,
			SrcNetworkName: source.NetworkName,
		},
		FQDN: fqdn,
	}
}

//MapKey generates a string which may be used to index an ordered pair of UniqueIPs. Concatenates IPs and UUIDs.
func (p UniqueSrcHostname) MapKey() string {
	var builder strings.Builder

	srcUUIDLen := 1 + len(p.SrcNetworkUUID.Data)

	builder.Grow(len(p.SrcIP) + srcUUIDLen + len(p.FQDN))
	builder.WriteString(p.SrcIP)
	builder.WriteByte(p.SrcNetworkUUID.Kind)
	builder.Write(p.SrcNetworkUUID.Data)

	builder.WriteString(p.FQDN)

	return builder.String()
}

// BSONKey generates a BSON map which may be used to index a given a unique
// src-fqdn pair
// Includes IP and Network UUID.
func (p UniqueSrcHostname) BSONKey() bson.M {
	key := bson.M{
		"src":              p.SrcIP,
		"src_network_uuid": p.SrcNetworkUUID,
		"fqdn":             p.FQDN,
	}
	return key
}
