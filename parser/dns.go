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
	// We don't filter out the src ips like we do with the conn
	// section since a c2 channel running over dns could have an
	// internal ip to internal ip connection and not having that ip
	// in the host table is limiting
	ignore := (filter.filterDomain(domain) || filter.filterSingleIP(srcIP))

	// If domain is not subject to filtering, process
	if ignore {
		return
	}

	// in some of these strings, the empty space will get counted as a domain,
	// don't add host or increment dns query count if queried domain
	// is blank or ends in 'in-addr.arpa'
	shouldUpdateHostInfo := (domain != "") && (!strings.HasSuffix(domain, "in-addr.arpa"))

	srcUniqIP := data.NewUniqueIP(srcIP, parseDNS.AgentUUID, parseDNS.AgentHostname)
	srcKey := srcUniqIP.MapKey()

	// ///////////////////////// CREATE EXPLODED DNS ENTRY /////////////////////////
	// increment domain map count for exploded dns
	{
		retVals.ExplodedDNSLock.Lock()
		retVals.ExplodedDNSMap[domain]++
		retVals.ExplodedDNSLock.Unlock()
	}

	// ///////////////////////// CREATE HOSTNAME ENTRY /////////////////////////
	// initialize the hostname input objects for new hostnames
	{
		retVals.HostnameLock.Lock()
		if _, ok := retVals.HostnameMap[domain]; !ok {
			retVals.HostnameMap[domain] = &hostname.Input{
				Host: domain,
			}
		}
		retVals.HostnameLock.Unlock()
	}

	// ///////////////////////// CREATE HOST ENTRY /////////////////////////
	if shouldUpdateHostInfo {
		retVals.HostLock.Lock()
		// Check if host map value is set, because this record could
		// come before a relevant conns record
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
	}

	// ///////////////////////// HOSTNAME UPDATES /////////////////////////
	{
		retVals.HostnameLock.Lock()
		// ///// UNION SOURCE HOST INTO HOSTNAME CLIENT SET /////
		retVals.HostnameMap[domain].ClientIPs.Insert(srcUniqIP)

		// ///// UNION HOST ANSWERS INTO HOSTNAME RESOLVED HOST SET /////
		if queryTypeName == "A" {
			for _, answer := range parseDNS.Answers {
				answerIP := net.ParseIP(answer)
				// Check if answer is an IP address and store it if it is
				if answerIP != nil {
					answerUniqIP := data.NewUniqueIP(
						answerIP, parseDNS.AgentUUID, parseDNS.AgentHostname,
					)
					retVals.HostnameMap[domain].ResolvedIPs.Insert(answerUniqIP)
				}
			}
		}
		retVals.HostnameLock.Unlock()
	}

	// ///////////////////////// HOST ENTRY UPDATES /////////////////////////
	if shouldUpdateHostInfo {
		retVals.HostLock.Lock()
		// ///// INCREMENT DNS QUERY COUNT FOR HOST /////
		// if there are no entries in the dnsquerycount map for this
		// srcKey, initialize map
		if retVals.HostMap[srcKey].DNSQueryCount == nil {
			retVals.HostMap[srcKey].DNSQueryCount = make(map[string]int64)
		}

		// increment the dns query count for this domain
		retVals.HostMap[srcKey].DNSQueryCount[domain]++
		retVals.HostLock.Unlock()
	}
}
