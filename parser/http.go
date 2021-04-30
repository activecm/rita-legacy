package parser

import (
	"net"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/beaconproxy"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/useragent"
	"github.com/activecm/rita/util"
)

func parseHTTPEntry(parseHTTP *parsetypes.HTTP, filter filter, retVals ParseResults) {
	// get source destination pair for connection record
	src := parseHTTP.Source
	dst := parseHTTP.Destination

	// parse addresses into binary format
	srcIP := net.ParseIP(src)
	dstIP := net.ParseIP(dst)

	// parse host
	fqdn := parseHTTP.Host

	if filter.filterDomain(fqdn) || filter.filterConnPair(srcIP, dstIP) {
		return
	}

	// disambiguate addresses which are not publicly routable
	srcUniqIP := data.NewUniqueIP(srcIP, parseHTTP.AgentUUID, parseHTTP.AgentHostname)
	dstUniqIP := data.NewUniqueIP(dstIP, parseHTTP.AgentUUID, parseHTTP.AgentHostname)
	srcProxyFQDNTrio := beaconproxy.NewUniqueSrcProxyHostnameTrio(srcUniqIP, dstUniqIP, fqdn)

	// get aggregation keys for ip addresses and connection pair
	srcProxyFQDNKey := srcProxyFQDNTrio.MapKey()

	// check if destination is a proxy server
	dstIsProxy := filter.checkIfProxyServer(dstIP)

	// parse method type
	method := parseHTTP.Method

	// check if internal IP is requesting a connection
	// through a proxy
	if method == "CONNECT" && dstIsProxy {

		// add client (src) IP to hostname map

		// Check if the map value is set
		retVals.ProxyUniqueConnLock.Lock()
		if _, ok := retVals.ProxyUniqueConnMap[srcProxyFQDNKey]; !ok {
			// create new host record with src and dst
			retVals.ProxyUniqueConnMap[srcProxyFQDNKey] = &beaconproxy.Input{
				Hosts: srcProxyFQDNTrio,
			}
		}
		retVals.ProxyUniqueConnLock.Unlock()

		// increment connection count
		retVals.ProxyUniqueConnLock.Lock()
		retVals.ProxyUniqueConnMap[srcProxyFQDNKey].ConnectionCount++
		retVals.ProxyUniqueConnLock.Unlock()

		// parse timestamp
		ts := parseHTTP.TimeStamp

		// add timestamp to unique timestamp list
		retVals.ProxyUniqueConnLock.Lock()
		if !util.Int64InSlice(ts, retVals.ProxyUniqueConnMap[srcProxyFQDNKey].TsList) {
			retVals.ProxyUniqueConnMap[srcProxyFQDNKey].TsList = append(
				retVals.ProxyUniqueConnMap[srcProxyFQDNKey].TsList, ts,
			)
		}
		retVals.ProxyUniqueConnLock.Unlock()
	}

	// parse out useragent info
	userAgentName := parseHTTP.UserAgent
	if userAgentName == "" {
		userAgentName = "Empty user agent string"
	}

	// create record if it doesn't exist
	retVals.UseragentLock.Lock()
	if _, ok := retVals.UseragentMap[userAgentName]; !ok {
		retVals.UseragentMap[userAgentName] = &useragent.Input{
			Name:     userAgentName,
			Seen:     1,
			Requests: []string{fqdn},
		}
		retVals.UseragentMap[userAgentName].OrigIps.Insert(srcUniqIP)
	} else {
		// increment times seen count
		retVals.UseragentMap[userAgentName].Seen++

		// add src of useragent request to unique array
		retVals.UseragentMap[userAgentName].OrigIps.Insert(srcUniqIP)

		// add request string to unique array
		if !util.StringInSlice(fqdn, retVals.UseragentMap[userAgentName].Requests) {
			retVals.UseragentMap[userAgentName].Requests = append(retVals.UseragentMap[userAgentName].Requests, fqdn)
		}
	}
	retVals.UseragentLock.Unlock()
}
