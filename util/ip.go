package util

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/globalsign/mgo/bson"
)

var privateIPBlocks []*net.IPNet

func init() {
	privateIPBlocks = ParseSubnets(
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

//ParseSubnets parses the provided subnets into net.ipnet format
func ParseSubnets(subnets []string) (parsedSubnets []*net.IPNet) {

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

//IPIsPubliclyRoutable checks if an IP address is publicly routable. See privateIPBlocks.
func IPIsPubliclyRoutable(ip net.IP) bool {
	// cache IPv4 conversion so it not performed every in every ip.IsXXX method
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	if ContainsIP(privateIPBlocks, ip) {
		return false
	}
	return true
}

//ContainsIP checks if a collection of subnets contains an IP
func ContainsIP(subnets []*net.IPNet, ip net.IP) bool {
	// cache IPv4 conversion so it not performed every in every Contains call
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}

	for _, block := range subnets {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

//ContainsDomain checks if a collection of domains contains an IP
func ContainsDomain(domains []string, host string) bool {

	for _, entry := range domains {

		// check for wildcard
		if strings.Contains(entry, "*") {

			// trim asterisk from the wildcard domain
			wildcardDomain := strings.TrimPrefix(entry, "*")

			//This would match a.mydomain.com, b.mydomain.com etc.,
			if strings.HasSuffix(host, wildcardDomain) {
				return true
			}

			// check match of top domain of wildcard
			// if a user added *.mydomain.com, this will include mydomain.com
			// in the filtering
			wildcardDomain = strings.TrimPrefix(wildcardDomain, ".")

			if host == wildcardDomain {
				return true
			}
		} else { // match on exact
			if host == entry {
				return true
			}
		}

	}
	return false
}

// IsIP returns true if string is a valid IP address
func IsIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

//IsIPv4 checks if an ip is ipv4
func IsIPv4(address string) bool {
	return strings.Count(address, ":") < 2
}

//IPv4ToBinary generates binary representations of the IPv4 addresses
func IPv4ToBinary(ipv4 net.IP) int64 {
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

//PublicNetworkName is the name bound to publicly routable UniqueIP addresses
const PublicNetworkName string = "Public"

//UnknownPrivateNetworkUUID ...
var UnknownPrivateNetworkUUID bson.Binary = bson.Binary{
	Kind: bson.BinaryUUID,
	Data: []byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe,
	},
}

//UnknownPrivateNetworkName ...
const UnknownPrivateNetworkName string = "Unknown Private"
