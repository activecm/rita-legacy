package parser

import (
	"net"
	"strings"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/sniconn"
	"github.com/activecm/rita/pkg/uconnproxy"
	"github.com/activecm/rita/pkg/useragent"

	log "github.com/sirupsen/logrus"
)

func parseHTTPEntry(parseHTTP *parsetypes.HTTP, filter filter, retVals ParseResults, logger *log.Logger) {
	// get source destination pair for connection record
	src := parseHTTP.Source
	dst := parseHTTP.Destination

	// parse addresses into binary format
	srcIP := net.ParseIP(src)
	dstIP := net.ParseIP(dst)

	// verify that both addresses were able to be parsed successfully
	if (srcIP == nil) || (dstIP == nil) {
		logger.WithFields(log.Fields{
			"uid": parseHTTP.UID,
			"src": parseHTTP.Source,
			"dst": parseHTTP.Destination,
		}).Error("Unable to parse valid ip address pair from http log entry, skipping entry.")
		return
	}

	// parse host
	fqdn := parseHTTP.Host

	// host field isn't always populated.
	// as a second option, parse out the host from the URI.
	// This isn't the first choice as it will take longer than
	// just grabbing the fqdn from the host field
	if fqdn == "" {
		uri := parseHTTP.URI

		minIndex := 0

		// handle if the URI has :// present (e.g., http://, https://, etc.)
		if protoIndex := strings.Index(uri, "://"); protoIndex != -1 {
			minIndex = protoIndex + len("://")
		}
		uri = uri[minIndex:]

		maxIndex := len(uri)
		if portIdx := strings.Index(uri, ":"); portIdx > -1 {
			// Case for if URI has the port number included (e.g., example.com:443).
			// This will also handle if the URI has a path appended as the path
			// appears after the port, so this will just lop off the path too.
			maxIndex = portIdx
		} else if pathIdx := strings.Index(uri, "/"); pathIdx > -1 {
			// Case for if the URI did not have a port but had a path
			// suffixed to it (e.g., example.com/somecoolpath
			maxIndex = pathIdx
		}
		uri = uri[:maxIndex]

		// at this point, the URI should be parsed down to just an FQDN
		fqdn = uri

	}

	// parse method type
	method := parseHTTP.Method

	// check if destination is a proxy server based on HTTP method
	dstIsProxy := (method == "CONNECT")

	// if the HTTP method is CONNECT, then the srcIP is communicating
	// to an FQDN through the dstIP proxy. We need to handle that
	// as a special case here so that we don't filter internal->internal
	// connections if the dstIP is an internal IP because the dstIP
	// is an intermediary and not the final destination.
	//
	// The dstIP filter check is not included for proxy connections either
	// because it isn't really the destination and I don't think that it makes
	// sense in this context to check for it. If the proxy IP is external,
	// this will also allow a user to filter results from other modules
	// (e.g., beacons), where false positives might arise due to the proxy IP
	// appearing as a destination, while still allowing for processing that
	// data for the proxy modules
	if dstIsProxy {
		if filter.filterDomain(fqdn) || filter.filterSingleIP(srcIP) {
			return
		}
		fqdnAsIPAddress := net.ParseIP(fqdn)
		if fqdnAsIPAddress != nil && filter.checkIfInternal(dstIP) && filter.filterConnPair(srcIP, fqdnAsIPAddress) {
			return
		}
	} else if filter.filterDomain(fqdn) || filter.filterConnPair(srcIP, dstIP) {
		return
	}

	// disambiguate addresses which are not publicly routable
	srcUniqIP := data.NewUniqueIP(srcIP, parseHTTP.AgentUUID, parseHTTP.AgentHostname)
	dstUniqIP := data.NewUniqueIP(dstIP, parseHTTP.AgentUUID, parseHTTP.AgentHostname)
	srcFQDNPair := data.NewUniqueSrcFQDNPair(srcUniqIP, fqdn)

	srcFQDNKey := srcFQDNPair.MapKey()

	updateUseragentsByHTTP(srcUniqIP, parseHTTP, retVals)

	// check if internal IP is requesting a connection through a proxy
	if dstIsProxy {
		updateProxiedUniqueConnectionsByHTTP(srcFQDNPair, dstUniqIP, parseHTTP, retVals)
		return
	}

	updateHTTPConnectionsByHTTP(srcIP, dstUniqIP, srcFQDNPair, srcFQDNKey, parseHTTP, filter, retVals)
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
			Name:     parseHTTP.UserAgent,
			OrigIps:  make(data.UniqueIPSet),
			Requests: make(data.StringSet),
		}
	}

	// ///// INCREMENT USERAGENT COUNTER /////
	retVals.UseragentMap[parseHTTP.UserAgent].Seen++

	// ///// UNION SOURCE HOST INTO USERAGENT ORIGINATING HOSTS /////
	retVals.UseragentMap[parseHTTP.UserAgent].OrigIps.Insert(srcUniqIP)

	// ///// UNION DESTINATION HOSTNAME INTO USERAGENT DESTINATIONS /////
	retVals.UseragentMap[parseHTTP.UserAgent].Requests.Insert(parseHTTP.Host)
}

func updateProxiedUniqueConnectionsByHTTP(srcFQDNPair data.UniqueSrcFQDNPair, dstUniqIP data.UniqueIP,
	parseHTTP *parsetypes.HTTP, retVals ParseResults) {

	retVals.ProxyUniqueConnLock.Lock()
	defer retVals.ProxyUniqueConnLock.Unlock()

	// get aggregation keys for src ip addresses and fqdn pair
	srcFQDNKey := srcFQDNPair.MapKey()

	if _, ok := retVals.ProxyUniqueConnMap[srcFQDNKey]; !ok {
		// create new host record with src and dst
		retVals.ProxyUniqueConnMap[srcFQDNKey] = &uconnproxy.Input{
			Hosts: srcFQDNPair,
			Proxy: dstUniqIP,
		}
	}

	// ///// INCREMENT THE CONNECTION COUNT FOR THE PROXIED UNIQUE CONNECTION /////
	retVals.ProxyUniqueConnMap[srcFQDNKey].ConnectionCount++

	// ///// APPEND TIMESTAMP TO PROXIED UNIQUE CONNECTION TIMESTAMP LIST /////
	ts := parseHTTP.TimeStamp

	retVals.ProxyUniqueConnMap[srcFQDNKey].TsList = append(
		retVals.ProxyUniqueConnMap[srcFQDNKey].TsList, ts,
	)
}

func updateHTTPConnectionsByHTTP(srcIP net.IP, dstUniqIP data.UniqueIP, srcFQDNPair data.UniqueSrcFQDNPair, srcFQDNKey string,
	parseHTTP *parsetypes.HTTP, filter filter, retVals ParseResults) {

	if len(srcFQDNPair.FQDN) == 0 {
		return // don't record HTTP connections when the FQDN is missing
	}

	retVals.HTTPConnLock.Lock()
	defer retVals.HTTPConnLock.Unlock()

	if _, ok := retVals.HTTPConnMap[srcFQDNKey]; !ok {
		inputVal := &sniconn.HTTPInput{
			Hosts:      srcFQDNPair,
			IsLocalSrc: filter.checkIfInternal(srcIP),

			Timestamps:      []int64{},
			RespondingIPs:   make(data.UniqueIPSet),
			RespondingPorts: make(data.IntSet),
			Methods:         make(data.StringSet),
			UserAgents:      make(data.StringSet),
		}

		retVals.HTTPConnMap[srcFQDNKey] = inputVal
	}

	// ///// INCREMENT THE CONNECTION COUNT FOR THE HTTP SNI CONNECTION /////
	retVals.HTTPConnMap[srcFQDNKey].ConnectionCount++

	// ///// APPEND TIMESTAMP TO HTTP TIMESTAMP LIST /////
	retVals.HTTPConnMap[srcFQDNKey].Timestamps = append(
		retVals.HTTPConnMap[srcFQDNKey].Timestamps, parseHTTP.TimeStamp,
	)

	// ///// UNION DESTINATION HOST INTO HTTP RESPONDING HOSTS /////
	retVals.HTTPConnMap[srcFQDNKey].RespondingIPs.Insert(dstUniqIP)

	// ///// UNION DESTINATION PORT INTO HTTP RESPONDING PORTS /////
	retVals.HTTPConnMap[srcFQDNKey].RespondingPorts.Insert(parseHTTP.DestinationPort)

	// ///// UNION METHOD INTO HTTP METHODS /////
	retVals.HTTPConnMap[srcFQDNKey].Methods.Insert(parseHTTP.Method)

	// ///// UNION USERAGENT INTO HTTP USERAGENTS /////
	retVals.HTTPConnMap[srcFQDNKey].UserAgents.Insert(parseHTTP.UserAgent)

	// ///// APPEND ZEEK RECORD UID INTO HTTP UID SET /////
	// This allows us to link conn record information to this
	// ip -> fqdn record such as data sizes.
	if len(parseHTTP.UID) > 0 {
		retVals.HTTPConnMap[srcFQDNKey].ZeekUIDs = append(
			retVals.HTTPConnMap[srcFQDNKey].ZeekUIDs,
			parseHTTP.UID,
		)
	}
}
