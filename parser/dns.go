package parser

import (
	"net"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/hostname"
)

func parseDNSEntry(parseDNS *parsetypes.DNS, filter filter, retVals ParseResults) {

	// extract and store the dns client ip address
	src := parseDNS.Source
	srcIP := net.ParseIP(src)
	dst := parseDNS.Destination
	dstIP := net.ParseIP(dst)

	// Run domain through filter to filter out certain domains and
	// filter out internal -> internal and external -> internal traffic
	//
	// If an internal DNS resolver is being used, make sure to add the IP address of the resolver to the filter's AlwaysInclude section
	ignore := (filter.filterDomain(parseDNS.Query) || filter.filterConnPair(srcIP, dstIP))

	// If domain is not subject to filtering, process
	if ignore {
		return
	}

	srcUniqIP := data.NewUniqueIP(srcIP, parseDNS.AgentUUID, parseDNS.AgentHostname)

	updateExplodedDNSbyDNS(parseDNS, retVals)
	updateHostnamesByDNS(srcUniqIP, parseDNS, retVals)
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
			Host:        parseDNS.Query,
			ClientIPs:   make(data.UniqueIPSet),
			ResolvedIPs: make(data.UniqueIPSet),
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
