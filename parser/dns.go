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
	domain := parseDNS.Query
	queryTypeName := parseDNS.QTypeName

	// extract and store the dns client ip address
	src := parseDNS.Source
	srcIP := net.ParseIP(src)

	// Run domain through filter to filter out certain domains
	ignore := (filter.filterDomain(domain) || filter.filterSingleIP(srcIP))

	// If domain is not subject to filtering, process
	if ignore {
		return
	}
	// increment domain map count for exploded dns
	retVals.ExplodedDNSLock.Lock()
	retVals.ExplodedDNSMap[domain]++
	retVals.ExplodedDNSLock.Unlock()

	// initialize the hostname input objects for new hostnames
	retVals.HostnameLock.Lock()
	if _, ok := retVals.HostnameMap[domain]; !ok {
		retVals.HostnameMap[domain] = &hostname.Input{
			Host: domain,
		}
	}
	retVals.HostnameLock.Unlock()

	srcUniqIP := data.NewUniqueIP(srcIP, parseDNS.AgentUUID, parseDNS.AgentHostname)
	srcKey := srcUniqIP.MapKey()

	retVals.HostnameLock.Lock()
	retVals.HostnameMap[domain].ClientIPs.Insert(srcUniqIP)
	retVals.HostnameLock.Unlock()

	if queryTypeName == "A" {
		answers := parseDNS.Answers
		for _, answer := range answers {
			answerIP := net.ParseIP(answer)
			// Check if answer is an IP address and store it if it is
			if answerIP != nil {
				answerUniqIP := data.NewUniqueIP(
					answerIP, parseDNS.AgentUUID, parseDNS.AgentHostname,
				)
				retVals.HostnameLock.Lock()
				retVals.HostnameMap[domain].ResolvedIPs.Insert(answerUniqIP)
				retVals.HostnameLock.Unlock()
			}
		}
	}

	// We don't filter out the src ips like we do with the conn
	// section since a c2 channel running over dns could have an
	// internal ip to internal ip connection and not having that ip
	// in the host table is limiting

	// in some of these strings, the empty space will get counted as a domain,
	// don't add host or increment dns query count if queried domain
	// is blank or ends in 'in-addr.arpa'
	if (domain != "") && (!strings.HasSuffix(domain, "in-addr.arpa")) {
		// Check if host map value is set, because this record could
		// come before a relevant conns record

		retVals.HostLock.Lock()
		if _, ok := retVals.HostMap[srcKey]; !ok {
			// create new uconn record with src and dst
			// Set IsLocalSrc and IsLocalDst fields based on InternalSubnets setting
			// we only need to do this once if the uconn record does not exist
			retVals.HostMap[srcKey] = &host.Input{
				Host:    srcUniqIP,
				IsLocal: filter.checkIfInternal(srcIP),
				IP4:     util.IsIPv4(src),
				IP4Bin:  util.IPv4ToBinary(srcIP),
			}
		}
		retVals.HostLock.Unlock()

		// if there are no entries in the dnsquerycount map for this
		// srcKey, initialize map
		retVals.HostLock.Lock()
		if retVals.HostMap[srcKey].DNSQueryCount == nil {
			retVals.HostMap[srcKey].DNSQueryCount = make(map[string]int64)
		}

		// increment the dns query count for this domain
		retVals.HostMap[srcKey].DNSQueryCount[domain]++
		retVals.HostLock.Unlock()
	}
}
