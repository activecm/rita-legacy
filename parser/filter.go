package parser

import (
	"fmt"
	"net"
	"os"
)

// filterConnPair returns true if a connection pair is filtered/excluded.
// This is determined by the following rules, in order:
//   1. Not filtered if either IP is on the AlwaysInclude list
//   2. Filtered if either IP is on the NeverInclude list
//   3. Not filtered if InternalSubnets is empty
//   4. Filtered if both IPs are internal or both are external
//   5. Not filtered in all other cases
func (fs *FSImporter) filterConnPair(src string, dst string) bool {
	// parse src and dst IPs
	srcIP := net.ParseIP(src)
	dstIP := net.ParseIP(dst)

	// check if on always included list
	isSrcIncluded := containsIP(fs.alwaysIncluded, srcIP)
	isDstIncluded := containsIP(fs.alwaysIncluded, dstIP)

	// check if on never included list
	isSrcExcluded := containsIP(fs.neverIncluded, srcIP)
	isDstExcluded := containsIP(fs.neverIncluded, dstIP)

	// if either IP is on the AlwaysInclude list, filter does not apply
	if isSrcIncluded || isDstIncluded {
		return false
	}

	// if either IP is on the NeverInclude list, filter applies
	if isSrcExcluded || isDstExcluded {
		return true
	}

	// if no internal subnets are defined, filter does not apply
	// this is was the default behavior before InternalSubnets was added
	if len(fs.internal) == 0 {
		return false
	}

	// check if src and dst are internal
	isSrcInternal := containsIP(fs.internal, srcIP)
	isDstInternal := containsIP(fs.internal, dstIP)

	// if both addresses are internal, filter applies
	if isSrcInternal && isDstInternal {
		return true
	}

	// if both addresses are external, filter applies
	if (!isSrcInternal) && (!isDstInternal) {
		return true
	}

	// default to not filter the connection pair
	return false
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
