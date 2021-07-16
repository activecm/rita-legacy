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

	updateUseragentsByHTTP(srcUniqIP, parseHTTP, retVals)

	// check if internal IP is requesting a connection through a proxy
	if parseHTTP.Method == "CONNECT" && filter.checkIfProxyServer(dstIP) {
		updateProxiedUniqueConnectionsByHTTP(srcProxyFQDNTrio, srcProxyFQDNKey, parseHTTP, retVals)
	}
}

func updateUseragentsByHTTP(srcUniqIP data.UniqueIP, parseHTTP *parsetypes.HTTP, retVals ParseResults) {

	retVals.UseragentLock.Lock()
	defer retVals.UseragentLock.Unlock()

	// parse out useragent info
	if parseHTTP.UserAgent == "" {
		parseHTTP.UserAgent = "Empty user agent string"
	}

	if _, ok := retVals.UseragentMap[parseHTTP.UserAgent]; !ok {
		retVals.UseragentMap[parseHTTP.UserAgent] = &useragent.Input{
			Name: parseHTTP.UserAgent,
		}
	}

	// ///// INCREMENT USERAGENT COUNTER /////
	retVals.UseragentMap[parseHTTP.UserAgent].Seen++

	// ///// UNION SOURCE HOST INTO USERAGENT ORIGINATING HOSTS /////
	retVals.UseragentMap[parseHTTP.UserAgent].OrigIps.Insert(srcUniqIP)

	// ///// UNION DESTINATION HOSTNAME INTO USERAGENT DESTINATIONS /////
	retVals.UseragentMap[parseHTTP.UserAgent].Requests.Insert(parseHTTP.Host)
}

func updateProxiedUniqueConnectionsByHTTP(srcProxyFQDNTrio beaconproxy.UniqueSrcProxyHostnameTrio, srcProxyFQDNKey string, parseHTTP *parsetypes.HTTP, retVals ParseResults) {

	retVals.ProxyUniqueConnLock.Lock()
	defer retVals.ProxyUniqueConnLock.Unlock()

	if _, ok := retVals.ProxyUniqueConnMap[srcProxyFQDNKey]; !ok {
		// create new host record with src and dst
		retVals.ProxyUniqueConnMap[srcProxyFQDNKey] = &beaconproxy.Input{
			Hosts: srcProxyFQDNTrio,
		}
	}

	// ///// INCREMENT THE CONNECTION COUNT FOR THE PROXIED UNIQUE CONNECTION /////
	retVals.ProxyUniqueConnMap[srcProxyFQDNKey].ConnectionCount++

	// ///// UNION TIMESTAMP WITH PROXIED UNIQUE CONNECTION TIMESTAMP SET /////
	ts := parseHTTP.TimeStamp
	if !util.Int64InSlice(ts, retVals.ProxyUniqueConnMap[srcProxyFQDNKey].TsList) {
		retVals.ProxyUniqueConnMap[srcProxyFQDNKey].TsList = append(
			retVals.ProxyUniqueConnMap[srcProxyFQDNKey].TsList, ts,
		)
	}
}
