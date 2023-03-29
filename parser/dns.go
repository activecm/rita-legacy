package parser

import (
	"net"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/hostname"

	log "github.com/sirupsen/logrus"
)

func parseDNSEntry(parseDNS *parsetypes.DNS, filter filter, retVals ParseResults, logger *log.Logger) {

	// extract and store the dns client ip address
	src := parseDNS.Source
	srcIP := net.ParseIP(src)

	// verify that ip address was parsed successfully
	if srcIP == nil {
		logger.WithFields(log.Fields{
			"uid": parseDNS.UID,
			"src": parseDNS.Source,
		}).Error("Unable to get valid client ip address from dns log entry, skipping entry.")
		return
	}

	// get domain
	domain := parseDNS.Query

	// verify that domain was parsed successfully
	if domain == "" {
		logger.WithFields(log.Fields{
			"uid":   parseDNS.UID,
			"query": parseDNS.Query,
		}).Error("Unable to parse valid domain from dns log entry, skipping entry.")
		return
	}

	// Run domain through filter to filter out certain domains
	// We don't filter out the src ips like we do with the conn
	// section since a c2 channel running over dns could have an
	// internal ip to internal ip connection and not having that ip
	// in the host table is limiting
	ignore := (filter.filterDomain(domain) || filter.filterSingleIP(srcIP))

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
