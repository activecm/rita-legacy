package parser

import (
	"encoding/binary"
	"errors"
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

var ErrNoAgentInfoSupplied = errors.New("could not uniquely identify local IP address without agent information")

//NewUniqueIP returns a new UniqueIP. The NetworkUUID and NetworkName fields are only filled if:
// - ip is not publicly routable
// - agentUUID and agentName are nonzero strings
// - agentUUID is a valid 128bit hex UUID
func newUniqueIP(ip net.IP, agentUUID, agentName string) (data.UniqueIP, error) {
	u := data.UniqueIP{}
	u.IP = ip.String()

	// don't set network uuid/ name if the ip is publicly routable
	if util.IPIsPubliclyRoutable(ip) {
		return u, nil
	}

	// only fill the network uuid/ name if they are valid
	if len(agentUUID) == 0 || len(agentName) == 0 {
		return u, ErrNoAgentInfoSupplied
	}
	id, err := uuid.Parse(agentUUID)
	if err != nil {
		return u, err
	}

	u.NetworkUUID = &bson.Binary{
		Kind: bson.BinaryUUID,
		Data: id[:],
	}
	u.NetworkName = &agentName
	return u, nil
}
