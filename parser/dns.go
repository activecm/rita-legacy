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
