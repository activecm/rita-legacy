package parser

import (
	"net"
	"strings"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/util"
)

func parseDNSEntry(parseDNS *parsetypes.DNS, filter filter, retVals ParseResults) {

	// extract and store the dns client ip address
	src := parseDNS.Source
	srcIP := net.ParseIP(src)

	// Run domain through filter to filter out certain domains
	// We don't filter out the src ips like we do with the conn
	// section since a c2 channel running over dns could have an
	// internal ip to internal ip connection and not having that ip
	// in the host table is limiting
	ignore := (filter.filterDomain(parseDNS.Query) || filter.filterSingleIP(srcIP))

	// If domain is not subject to filtering, process
	if ignore {
		return
	}

	srcUniqIP := data.NewUniqueIP(srcIP, parseDNS.AgentUUID, parseDNS.AgentHostname)
	srcKey := srcUniqIP.MapKey()

	updateExplodedDNSbyDNS(parseDNS, retVals)
	updateHostnamesByDNS(srcUniqIP, parseDNS, retVals)

	// in some of these strings, the empty space will get counted as a domain,
	// don't add host or increment dns query count if queried domain
	// is blank or ends in 'in-addr.arpa'
	if (parseDNS.Query != "") && (!strings.HasSuffix(parseDNS.Query, "in-addr.arpa")) {
		updateHostsByDNS(srcIP, srcUniqIP, srcKey, parseDNS, filter, retVals)
	}
}

func updateExplodedDNSbyDNS(parseDNS *parsetypes.DNS, retVals ParseResults) {

	retVals.ExplodedDNSLock.Lock()
	defer retVals.ExplodedDNSLock.Unlock()

	retVals.ExplodedDNSMap[parseDNS.Query]++
}

func updateHostnamesByDNS(srcUniqIP data.UniqueIP, parseDNS *parsetypes.DNS, retVals ParseResults) {

	retVals.HostnameLock.Lock()
	defer retVals.HostnameLock.Unlock()

	if _, ok := retVals.HostnameMap[parseDNS.Query]; !ok {
		retVals.HostnameMap[parseDNS.Query] = &hostname.Input{
			Host: parseDNS.Query,
		}
	}

	// ///// UNION SOURCE HOST INTO HOSTNAME CLIENT SET /////
	retVals.HostnameMap[parseDNS.Query].ClientIPs.Insert(srcUniqIP)

	// ///// UNION HOST ANSWERS INTO HOSTNAME RESOLVED HOST SET /////
	if parseDNS.QTypeName == "A" {
		for _, answer := range parseDNS.Answers {
			answerIP := net.ParseIP(answer)
			// Check if answer is an IP address and store it if it is
			if answerIP != nil {
				answerUniqIP := data.NewUniqueIP(answerIP, parseDNS.AgentUUID, parseDNS.AgentHostname)
				retVals.HostnameMap[parseDNS.Query].ResolvedIPs.Insert(answerUniqIP)
			}
		}
	}
}

func updateHostsByDNS(srcIP net.IP, srcUniqIP data.UniqueIP, srcKey string,
	parseDNS *parsetypes.DNS, filter filter, retVals ParseResults) {

	retVals.HostLock.Lock()
	defer retVals.HostLock.Unlock()

	// Check if host map value is set, because this record could
	// come before a relevant conns record
	if _, ok := retVals.HostMap[srcKey]; !ok {
		// create new uconn record with src and dst
		// Set IsLocalSrc and IsLocalDst fields based on InternalSubnets setting
		// we only need to do this once if the uconn record does not exist
		retVals.HostMap[srcKey] = &host.Input{
			Host:    srcUniqIP,
			IsLocal: filter.checkIfInternal(srcIP),
			IP4:     util.IsIPv4(srcUniqIP.IP),
			IP4Bin:  util.IPv4ToBinary(srcIP),
		}
	}

	// ///// INCREMENT DNS QUERY COUNT FOR HOST /////
	// if there are no entries in the dnsquerycount map for this
	// srcKey, initialize map
	if retVals.HostMap[srcKey].DNSQueryCount == nil {
		retVals.HostMap[srcKey].DNSQueryCount = make(map[string]int64)
	}

	// increment the dns query count for this domain
	retVals.HostMap[srcKey].DNSQueryCount[parseDNS.Query]++

}
