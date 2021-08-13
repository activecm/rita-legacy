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

//UniqueSrcProxyHostnameTrio is used to make a tuple of
// Src IP/UUID/Name, proxy server IP/UUID/Name (UniqueDstIP), and an FQDN
// to which the Src IP was attempting to communicate
type UniqueSrcProxyHostnameTrio struct {
	data.UniqueSrcIP `bson:",inline"`
	data.UniqueDstIP `bson:",inline"`
	FQDN             string `bson:"fqdn"`
}

//Input structure for sending data
//to the analyzer. Contains a tuple of
// Src IP/UUID/Name, Dst IP/UUID/Name (intermediary proxy server)
// and the FQDN to which the Src was attempting to connect. Contains
// a list of unique time stamps for the connections out from the Src to
// the FQDN via the proxy server and a count of the connections.
type Input struct {
	Hosts           UniqueSrcProxyHostnameTrio
	TsList          []int64
	ConnectionCount int64
}

//NewUniqueSrcProxyHostnameTrio binds a pair of UniqueIPs where direction matters.
func NewUniqueSrcProxyHostnameTrio(source data.UniqueIP, proxy data.UniqueIP, fqdn string) UniqueSrcProxyHostnameTrio {
	return UniqueSrcProxyHostnameTrio{
		UniqueSrcIP: data.UniqueSrcIP{
			SrcIP:          source.IP,
			SrcNetworkUUID: source.NetworkUUID,
			SrcNetworkName: source.NetworkName,
		},
		UniqueDstIP: data.UniqueDstIP{
			DstIP:          proxy.IP,
			DstNetworkUUID: proxy.NetworkUUID,
			DstNetworkName: proxy.NetworkName,
		},
		FQDN: fqdn,
	}
}

//MapKey generates a string which may be used to index an ordered pair of UniqueIPs. Concatenates IPs and UUIDs.
func (p UniqueSrcProxyHostnameTrio) MapKey() string {
	var builder strings.Builder

	srcUUIDLen := 1 + len(p.SrcNetworkUUID.Data)
	proxyUUIDLen := 1 + len(p.DstNetworkUUID.Data)

	builder.Grow(len(p.SrcIP) + srcUUIDLen + len(p.DstIP) + len(p.FQDN) + proxyUUIDLen)
	builder.WriteString(p.SrcIP)
	builder.WriteString(p.DstIP)
	builder.WriteByte(p.SrcNetworkUUID.Kind)
	builder.Write(p.SrcNetworkUUID.Data)
	builder.WriteByte(p.DstNetworkUUID.Kind)
	builder.Write(p.DstNetworkUUID.Data)

	builder.WriteString(p.FQDN)

	return builder.String()
}

// BSONKey generates a BSON map which may be used to index a given a unique
// src-dst-fqdn pair
// Includes IP and Network UUID.
func (p UniqueSrcProxyHostnameTrio) BSONKey() bson.M {
	key := bson.M{
		"src":              p.SrcIP,
		"src_network_uuid": p.SrcNetworkUUID,
		"dst":              p.DstIP,
		"dst_network_uuid": p.DstNetworkUUID,
		"fqdn":             p.FQDN,
	}
	return key
}
