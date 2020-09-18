package parser

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
	"github.com/google/uuid"
)

var privateIPBlocks []*net.IPNet

func init() {
	privateIPBlocks = getParsedSubnets(
		[]string{
			//"127.0.0.0/8",    // IPv4 Loopback; handled by ip.IsLoopback
			//"::1/128",        // IPv6 Loopback; handled by ip.IsLoopback
			//"169.254.0.0/16", // RFC3927 link-local; handled by ip.IsLinkLocalUnicast()
			//"fe80::/10",      // IPv6 link-local; handled by ip.IsLinkLocalUnicast()
			"10.0.0.0/8",     // RFC1918
			"172.16.0.0/12",  // RFC1918
			"192.168.0.0/16", // RFC1918
			"fc00::/7",       // IPv6 unique local addr
		})
}

//ipIsPubliclyRoutable checks if an IP address is publicly routable. See privateIPBlocks.
func ipIsPubliclyRoutable(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	if containsIP(privateIPBlocks, ip) {
		return false
	}
	return true
}

//parseSubnets parses the provided subnets into net.ipnet format
func getParsedSubnets(subnets []string) (parsedSubnets []*net.IPNet) {

	for _, entry := range subnets {
		//try to parse out cidr range
		_, block, err := net.ParseCIDR(entry)

		//if there was an error, check if entry was an IP not a range
		if err != nil {
			// try to parse out IP as range of single host
			_, block, err = net.ParseCIDR(entry + "/32")

			// if error, report and return
			if err != nil {
				fmt.Fprintf(os.Stdout, "Error parsing CIDR entry: %s\n", err.Error())
				os.Exit(-1)
				return
			}
		}

		// add cidr range to list
		parsedSubnets = append(parsedSubnets, block)
	}
	return
}

//containsIP checks if a specified subnet contains an ip
func containsIP(subnets []*net.IPNet, ip net.IP) bool {
	for _, block := range subnets {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

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
	if ipIsPubliclyRoutable(ip) {
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
