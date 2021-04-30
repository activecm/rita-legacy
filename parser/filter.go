package parser

import (
	"net"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/util"
)

// filter provides methods for excluding IP addresses, domains, and determining proxy servers during the import step
// based on the user configuration
type filter struct {
	internal         []*net.IPNet
	httpProxyServers []*net.IPNet
	alwaysIncluded   []*net.IPNet
	neverIncluded    []*net.IPNet

	alwaysIncludedDomain []string
	neverIncludedDomain  []string
}

func newFilter(conf *config.Config) filter {
	return filter{
		internal:             util.ParseSubnets(conf.S.Filtering.InternalSubnets),
		httpProxyServers:     util.ParseSubnets(conf.S.Filtering.HTTPProxyServers),
		alwaysIncluded:       util.ParseSubnets(conf.S.Filtering.AlwaysInclude),
		neverIncluded:        util.ParseSubnets(conf.S.Filtering.NeverInclude),
		alwaysIncludedDomain: conf.S.Filtering.AlwaysIncludeDomain,
		neverIncludedDomain:  conf.S.Filtering.NeverIncludeDomain,
	}
}

// filterConnPair returns true if a connection pair is filtered/excluded.
// This is determined by the following rules, in order:
//   1. Not filtered if either IP is on the AlwaysInclude list
//   2. Filtered if either IP is on the NeverInclude list
//   3. Not filtered if InternalSubnets is empty
//   4. Filtered if both IPs are internal or both are external
//   5. Not filtered in all other cases
func (fs *filter) filterConnPair(srcIP net.IP, dstIP net.IP) bool {
	// check if on always included list
	isSrcIncluded := util.ContainsIP(fs.alwaysIncluded, srcIP)
	isDstIncluded := util.ContainsIP(fs.alwaysIncluded, dstIP)

	// check if on never included list
	isSrcExcluded := util.ContainsIP(fs.neverIncluded, srcIP)
	isDstExcluded := util.ContainsIP(fs.neverIncluded, dstIP)

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
	isSrcInternal := util.ContainsIP(fs.internal, srcIP)
	isDstInternal := util.ContainsIP(fs.internal, dstIP)

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

// filterSingleIP returns true if an IP is filtered/excluded.
// This is determined by the following rules, in order:
//   1. Not filtered IP is on the AlwaysInclude list
//   2. Filtered IP is on the NeverInclude list
//   3. Not filtered in all other cases
func (fs *filter) filterSingleIP(IP net.IP) bool {
	// check if on always included list
	if util.ContainsIP(fs.alwaysIncluded, IP) {
		return false
	}

	// check if on never included list
	if util.ContainsIP(fs.neverIncluded, IP) {
		return true
	}

	// default to not filter the IP address
	return false
}

// filterDomain returns true if a domain is filtered/excluded.
// This is determined by the following rules, in order:
//   1. Not filtered if domain is on the AlwaysInclude list
//   2. Filtered if domain is on the NeverInclude list
//   5. Not filtered in all other cases
func (fs *filter) filterDomain(domain string) bool {
	// check if on always included list
	isDomainIncluded := util.ContainsDomain(fs.alwaysIncludedDomain, domain)

	// check if on never included list
	isDomainExcluded := util.ContainsDomain(fs.neverIncludedDomain, domain)

	// if either IP is on the AlwaysInclude list, filter does not apply
	if isDomainIncluded {
		return false
	}

	// if either IP is on the NeverInclude list, filter applies
	if isDomainExcluded {
		return true
	}

	// default to not filter the connection pair
	return false
}

func (fs *filter) checkIfInternal(host net.IP) bool {
	return util.ContainsIP(fs.internal, host)
}

func (fs *filter) checkIfProxyServer(host net.IP) bool {
	return util.ContainsIP(fs.httpProxyServers, host)
}
