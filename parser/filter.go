package parser

import (
	"fmt"
	"net"

	"github.com/activecm/rita/resources"
)

func (fs *FSImporter) filterConnPair(src string, dst string) (ignore bool) {
	// default is not to ignore connection pair
	ignore = false

	// parse src and dst IPs
	srcIP := net.ParseIP(src)
	dstIP := net.ParseIP(dst)

	// check if on always included list
	isSrcIncluded := fs.isAlwaysIncluded(srcIP)
	isDstIncluded := fs.isAlwaysIncluded(dstIP)

	// check if on never included list
	isSrcExcluded := fs.isNeverIncluded(srcIP)
	isDstExcluded := fs.isNeverIncluded(dstIP)

	// if a result is on both lists we don't ignore it (need error handling later)
	if (isSrcIncluded && isSrcExcluded) || (isDstIncluded && isDstExcluded) {
		ignore = false
		return
	}

	// check if one of the addresses should never be excluded from results
	if isSrcExcluded || isDstExcluded {
		ignore = true
		return
	}

	// check if one of the addresses should never be excluded from results
	if isSrcIncluded || isDstIncluded {
		ignore = false
		return
	}

	// verify if src and dst are internal or external
	isSrcInternal := fs.isInternalAddress(srcIP)
	isDstInternal := fs.isInternalAddress(dstIP)

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

// Get internal subnets from the config file
// todo: Error if a valid CIDR is not provided
func getInternalSubnets(res *resources.Resources) []*net.IPNet {
	var internalIPSubnets []*net.IPNet

	internalFilters := res.Config.S.Filtering.InternalSubnets
	for _, cidr := range internalFilters {
		_, block, err := net.ParseCIDR(cidr)
		internalIPSubnets = append(internalIPSubnets, block)
		if err != nil {
			fmt.Println("Error parsing config file CIDR.")
			fmt.Println(err)
		}
	}
	return internalIPSubnets
}

// Get "always included" subnets from the config file
func getIncludedSubnets(res *resources.Resources) (includedSubnets []*net.IPNet) {

	alwaysIncluded := res.Config.S.Filtering.AlwaysInclude

	for _, entry := range alwaysIncluded {
		//try to parse out cidr range
		_, block, err := net.ParseCIDR(entry)

		//if there was an error, check if entry was an IP not a range
		if err != nil {
			// try to parse out IP as range of single host
			_, block, err = net.ParseCIDR(entry + "/32")

			// if error, report and return
			if err != nil {
				fmt.Println("Error parsing config file entry.")
				fmt.Println(err)
				return
			}
		}

		// add cidr range to list
		includedSubnets = append(includedSubnets, block)
	}
	return
}

// Get "always included" subnets from the config file
func getExcludedSubnets(res *resources.Resources) (excludedSubnets []*net.IPNet) {

	neverInclude := res.Config.S.Filtering.NeverInclude

	for _, entry := range neverInclude {
		//try to parse out cidr range
		_, block, err := net.ParseCIDR(entry)

		//if there was an error, check if entry was an IP not a range
		if err != nil {
			// try to parse out IP as range of single host
			_, block, err = net.ParseCIDR(entry + "/32")

			// if error, report and return
			if err != nil {
				fmt.Println("Error parsing config file entry.")
				fmt.Println(err)
				return
			}
		}

		// add cidr range to list
		excludedSubnets = append(excludedSubnets, block)
	}
	return
}

// Check if a single IP address is internal
func (fs *FSImporter) isInternalAddress(ip net.IP) bool {
	for _, block := range fs.internal {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// Check if a single IP address should always be included
func (fs *FSImporter) isAlwaysIncluded(ip net.IP) bool {
	for _, block := range fs.alwaysIncluded {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// Check if a single IP address should always be included
func (fs *FSImporter) isNeverIncluded(ip net.IP) bool {
	for _, block := range fs.neverIncluded {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
