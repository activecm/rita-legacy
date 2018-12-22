package parser

import (
	"fmt"
	"net"
	"os"
)

func (fs *FSImporter) filterConnPair(src string, dst string) (ignore bool) {
	// default is not to ignore connection pair
	ignore = false

	// parse src and dst IPs
	srcIP := net.ParseIP(src)
	dstIP := net.ParseIP(dst)

	// check if on always included list
	isSrcIncluded := containsIP(fs.alwaysIncluded, srcIP)
	isDstIncluded := containsIP(fs.alwaysIncluded, dstIP)

	// check if on never included list
	isSrcExcluded := containsIP(fs.neverIncluded, srcIP)
	isDstExcluded := containsIP(fs.neverIncluded, dstIP)

	// check if one of the addresses should never be excluded from results
	if isSrcIncluded || isDstIncluded {
		ignore = false
		return
	}

	// check if one of the addresses should be excluded from results
	if isSrcExcluded || isDstExcluded {
		ignore = true
		return
	}

	// verify if src and dst are internal or external
	isSrcInternal := containsIP(fs.internal, srcIP)
	isDstInternal := containsIP(fs.internal, dstIP)

	// if both addresses are internal, filter applies
	if isSrcInternal && isDstInternal {
		ignore = true
		return
	}

	// if both addresses are external, filter applies
	if (!isSrcInternal) && (!isDstInternal) {
		ignore = true
		return
	}

	return
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
