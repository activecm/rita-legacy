package parser

import (
	"encoding/binary"
	"net"
	"strings"

	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

//isIPv4 checks if an ip is ipv4
func isIPv4(address string) bool {
	return strings.Count(address, ":") < 2
}

//ipv4ToBinary generates binary representations of the IPv4 addresses
func ipv4ToBinary(ipv4 net.IP) int64 {
	return int64(binary.BigEndian.Uint32(ipv4[12:16]))
}

//PublicNetworkUUID is the UUID bound to publicly routable UniqueIP addresses
var PublicNetworkUUID bson.Binary = bson.Binary{
	Kind: bson.BinaryUUID,
	Data: []byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	},
}

const PublicNetworkName string = "Public"

var UnknownPrivateNetworkUUID bson.Binary = bson.Binary{
	Kind: bson.BinaryUUID,
	Data: []byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe,
	},
}

const UnknownPrivateNetworkName string = "Unknown Private"

//newUniqueIP returns a new UniqueIP. If the given ip is publicly routable, the resulting UniqueIP's
//NetworkUUID and NetworkName will be set to PublicNetworkUUID and PublicNetworkName respectively.
//Otherwise, the NetworkUUID and NetworkName will be set based on the provided agentName and agentUUID.
//If the provided agent data is invalid, the NetworkUUID and NetworkName will be set to
//UnknownPrivateNetworkUUID and UnknownPrivateNetworkName.
func newUniqueIP(ip net.IP, agentUUID, agentName string) data.UniqueIP {
	u := data.UniqueIP{}
	u.IP = ip.String()

	if util.IPIsPubliclyRoutable(ip) {
		u.NetworkName = PublicNetworkName
		u.NetworkUUID = PublicNetworkUUID
		return u
	}

	id, err := uuid.Parse(agentUUID)
	if err != nil || len(agentUUID) == 0 || len(agentName) == 0 {
		u.NetworkName = UnknownPrivateNetworkName
		u.NetworkUUID = UnknownPrivateNetworkUUID
		return u
	}

	u.NetworkUUID = bson.Binary{
		Kind: bson.BinaryUUID,
		Data: id[:],
	}
	u.NetworkName = agentName
	return u
}
