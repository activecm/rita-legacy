package parser

import (
	"net"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/hostname"

	log "github.com/sirupsen/logrus"
)

func parseDNSEntry(parseDNS *parsetypes.DNS, filter filter, retVals ParseResults, logger *log.Logger) {
	// get source destination pair
	src := parseDNS.Source
	dst := parseDNS.Destination

	// parse addresses into binary format
	srcIP := net.ParseIP(src)
	dstIP := net.ParseIP(dst)

	// verify that both addresses were able to be parsed successfully
	if (srcIP == nil) || (dstIP == nil) {
		logger.WithFields(log.Fields{
			"uid": parseDNS.UID,
			"src": parseDNS.Source,
			"dst": parseDNS.Destination,
		}).Error("Unable to parse valid ip address pair from dns log entry, skipping entry.")
		return
	}

	// get domain
	domain := parseDNS.Query

	// Run domain through filter to filter out certain domains and
	// filter out traffic which is external -> external or external -> internal (if specified in the config file)
	ignore := (filter.filterDomain(domain) || filter.filterDNSPair(srcIP, dstIP))

	// If domain is not subject to filtering, process
	if ignore {
		return
	}

	srcUniqIP := data.NewUniqueIP(srcIP, parseDNS.AgentUUID, parseDNS.AgentHostname)

	updateExplodedDNSbyDNS(domain, retVals)
	updateHostnamesByDNS(srcUniqIP, domain, parseDNS, retVals)
}

func updateExplodedDNSbyDNS(domain string, retVals ParseResults) {

	retVals.ExplodedDNSLock.Lock()
	defer retVals.ExplodedDNSLock.Unlock()

	retVals.ExplodedDNSMap[domain]++
}

func updateHostnamesByDNS(srcUniqIP data.UniqueIP, domain string, parseDNS *parsetypes.DNS, retVals ParseResults) {

	retVals.HostnameLock.Lock()
	defer retVals.HostnameLock.Unlock()

	if _, ok := retVals.HostnameMap[domain]; !ok {
		retVals.HostnameMap[domain] = &hostname.Input{
			Host:        domain,
			ClientIPs:   make(data.UniqueIPSet),
			ResolvedIPs: make(data.UniqueIPSet),
		}
	}

	// ///// UNION SOURCE HOST INTO HOSTNAME CLIENT SET /////
	retVals.HostnameMap[domain].ClientIPs.Insert(srcUniqIP)

	// ///// UNION HOST ANSWERS INTO HOSTNAME RESOLVED HOST SET /////
	if parseDNS.QTypeName == "A" {
		for _, answer := range parseDNS.Answers {
			answerIP := net.ParseIP(answer)
			// Check if answer is an IP address and store it if it is
			if answerIP != nil {
				answerUniqIP := data.NewUniqueIP(answerIP, parseDNS.AgentUUID, parseDNS.AgentHostname)
				retVals.HostnameMap[domain].ResolvedIPs.Insert(answerUniqIP)
			}
		}
	}
}
