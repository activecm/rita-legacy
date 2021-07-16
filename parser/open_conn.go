package parser

import (
	"math"
	"net"
	"strconv"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/util"
)

func parseOpenConnEntry(parseConn *parsetypes.OpenConn, filter filter, retVals ParseResults) {
	// get source destination pair for connection record
	src := parseConn.Source
	dst := parseConn.Destination

	// parse addresses into binary format
	srcIP := net.ParseIP(src)
	dstIP := net.ParseIP(dst)

	// Run conn pair through filter to filter out certain connections
	ignore := filter.filterConnPair(srcIP, dstIP)

	// If connection pair is not subject to filtering, process
	if ignore {
		return
	}

	// disambiConnguate addresses which are not publicly routable
	srcUniqIP := data.NewUniqueIP(srcIP, parseConn.AgentUUID, parseConn.AgentHostname)
	dstUniqIP := data.NewUniqueIP(dstIP, parseConn.AgentUUID, parseConn.AgentHostname)
	srcDstPair := data.NewUniqueIPPair(srcUniqIP, dstUniqIP)

	// get aggregation keys for ip addresses and connection pair
	srcKey := srcUniqIP.MapKey()
	dstKey := dstUniqIP.MapKey()
	srcDstKey := srcDstPair.MapKey()

	roundedDuration := math.Ceil(parseConn.Duration*10000) / 10000
	twoWayIPBytes := int64(parseConn.OrigIPBytes + parseConn.RespIPBytes)

	var tuple string
	if parseConn.Service == "" {
		tuple = strconv.Itoa(parseConn.DestinationPort) + ":" + parseConn.Proto + ":-"
	} else {
		tuple = strconv.Itoa(parseConn.DestinationPort) + ":" + parseConn.Proto + ":" + parseConn.Service
	}

	newUniqueConnection, setUPPSFlag := updateUniqueConnectionsByOpenConn(
		srcIP, dstIP, srcDstPair, srcDstKey, roundedDuration, twoWayIPBytes, tuple, parseConn, filter, retVals,
	)

	updateHostsByOpenConn(
		srcIP, dstIP, srcUniqIP, dstUniqIP, srcKey, dstKey, newUniqueConnection, setUPPSFlag,
		roundedDuration, twoWayIPBytes, tuple, parseConn, filter, retVals,
	)

	updateCertificatesByOpenConn(dstKey, tuple, retVals)

}

func updateUniqueConnectionsByOpenConn(srcIP, dstIP net.IP, srcDstPair data.UniqueIPPair, srcDstKey string,
	roundedDuration float64, twoWayIPBytes int64, tuple string,
	parseConn *parsetypes.OpenConn, filter filter, retVals ParseResults) (newEntry bool, setUPPSFlag bool) {

	retVals.UniqueConnLock.Lock()
	defer retVals.UniqueConnLock.Unlock()

	setUPPSFlag = false
	newEntry = false

	var uconnExists bool
	if _, uconnExists = retVals.UniqueConnMap[srcDstKey]; !uconnExists {
		newEntry = true

		// create new uconn record with src and dst
		// Set IsLocalSrc and IsLocalDst fields based on InternalSubnets setting
		// we only need to do this once if the uconn record does not exist
		retVals.UniqueConnMap[srcDstKey] = &uconn.Input{
			Hosts:      srcDstPair,
			IsLocalSrc: filter.checkIfInternal(srcIP),
			IsLocalDst: filter.checkIfInternal(dstIP),
		}
	}

	// ///// SET UNEXPECTED (PORT PROTOCOL SERVICE) FLAG /////
	// this is to keep track of how many times a host connected to
	// an unexpected port - proto - service Tuple
	// we only want to increment the count once per unique destination,
	// not once per connection, hence the flag and the check
	if !retVals.UniqueConnMap[srcDstKey].UPPSFlag {
		for _, entry := range trustedAppReferenceList {
			if (parseConn.Proto == entry.protocol) && (parseConn.DestinationPort == entry.port) &&
				(parseConn.Service != entry.service) {
				setUPPSFlag = true
				retVals.UniqueConnMap[srcDstKey].UPPSFlag = true
				break
			}
		}
	}

	// ///// MARK UNIQUE CONNECTION AS OPEN /////
	// If the ConnStateList map doesn't exist for this entry, create it.
	if retVals.UniqueConnMap[srcDstKey].ConnStateMap == nil {
		retVals.UniqueConnMap[srcDstKey].ConnStateMap = make(map[string]*uconn.ConnState)
	}

	// If an entry for this open connection is present, first check
	// to make sure it hasn't been marked as closed. If it was marked
	// as closed, that supersedes any open connections information.
	// Otherwise, if it's open then check if this more up-to-date data
	// (i.e., longer duration)
	if _, ok := retVals.UniqueConnMap[srcDstKey].ConnStateMap[parseConn.UID]; ok {
		if retVals.UniqueConnMap[srcDstKey].ConnStateMap[parseConn.UID].Open &&
			roundedDuration > retVals.UniqueConnMap[srcDstKey].ConnStateMap[parseConn.UID].Duration {

			// If current duration is longer than previous duration, we can
			// also assume that current bytes is /at least/ as big as the
			// stored value for bytes...same for OrigBytes
			retVals.UniqueConnMap[srcDstKey].ConnStateMap[parseConn.UID].Duration = roundedDuration
			retVals.UniqueConnMap[srcDstKey].ConnStateMap[parseConn.UID].Bytes = twoWayIPBytes
			retVals.UniqueConnMap[srcDstKey].ConnStateMap[parseConn.UID].OrigBytes = parseConn.OrigBytes
		}
	} else {
		// No entry was present for a connection with this UID. Create a new
		// entry and set the Open state accordingly
		retVals.UniqueConnMap[srcDstKey].ConnStateMap[parseConn.UID] = &uconn.ConnState{
			Bytes:     twoWayIPBytes,
			Duration:  roundedDuration,
			Open:      true,
			OrigBytes: parseConn.OrigBytes,
			Ts:        parseConn.TimeStamp, //ts is the timestamp at which the connection was detected
			Tuple:     tuple,
		}
	}

	// ///// UNION (PORT PROTOCOL SERVICE) TUPLE INTO SET FOR UNIQUE CONNECTION /////
	retVals.UniqueConnMap[srcDstKey].Tuples.Insert(tuple)

	// ///// DETERMINE THE LONGEST DURATION SEEN FOR THIS UNIQUE CONNECTION /////
	// Replace existing duration if current duration is higher
	if roundedDuration > retVals.UniqueConnMap[srcDstKey].MaxDuration {
		retVals.UniqueConnMap[srcDstKey].MaxDuration = roundedDuration
	}

	// NOTE: We are not incrementing uconn.ConnectionCount until the
	// connection closes to prevent double-counting.
	return
}

func updateHostsByOpenConn(srcIP, dstIP net.IP, srcUniqIP, dstUniqIP data.UniqueIP, srcKey, dstKey string,
	newUniqueConnection, setUPPSFlag bool, roundedDuration float64, twoWayIPBytes int64, tuple string,
	parseConn *parsetypes.OpenConn, filter filter, retVals ParseResults) {

	retVals.HostLock.Lock()
	defer retVals.HostLock.Unlock()

	if _, ok := retVals.HostMap[srcKey]; !ok {
		// create new host record with src and dst
		retVals.HostMap[srcKey] = &host.Input{
			Host:    srcUniqIP,
			IsLocal: filter.checkIfInternal(srcIP),
			IP4:     util.IsIPv4(srcUniqIP.IP),
			IP4Bin:  util.IPv4ToBinary(srcIP),
		}
	}

	// Check if the map value is set
	if _, ok := retVals.HostMap[dstKey]; !ok {
		// create new host record with src and dst
		retVals.HostMap[dstKey] = &host.Input{
			Host:    dstUniqIP,
			IsLocal: filter.checkIfInternal(dstIP),
			IP4:     util.IsIPv4(dstUniqIP.IP),
			IP4Bin:  util.IPv4ToBinary(dstIP),
		}
	}

	// ///// INCREMENT SOURCE / DESTINATION COUNTERS FOR HOSTS /////
	// We only want to do this once for each unique connection entry
	// Can do this even if the connection is open. For each set of logs we
	// process, we increment these values once per unique connection.
	// This might mean that an open connection has caused these values
	// to be incremented, but that is ok.
	if newUniqueConnection {
		retVals.HostMap[srcKey].CountSrc++
		retVals.HostMap[dstKey].CountDst++
	}

	// ///// INCREMENT HOST UNEXPECTED (PORT PROTOCOL SERVICE) COUNTER /////
	// only do this once per flagged unique connection
	// similar to incrementing the host connection counts, this counter is only
	// increased once per new unique connection with a UPPS tuple
	if setUPPSFlag {
		retVals.HostMap[srcKey].UntrustedAppConnCount++
	}

	// NOTE: The max duration is not accumulated into the total duration
	// until the connection closes out

	// ///// DETERMINE THE LONGEST DURATION SEEN FOR THE SOURCE HOST /////
	if roundedDuration > retVals.HostMap[srcKey].MaxDuration {
		retVals.HostMap[srcKey].MaxDuration = roundedDuration
	}

	// ///// DETERMINE THE LONGEST DURATION SEEN FOR THE DESTINATION HOST /////
	if roundedDuration > retVals.HostMap[dstKey].MaxDuration {
		retVals.HostMap[dstKey].MaxDuration = roundedDuration
	}

	// NOTE: We are not incrementing host.ConnectionCount until the
	// connection closes to prevent double-counting.
}

func updateCertificatesByOpenConn(dstKey string, tuple string, retVals ParseResults) {

	retVals.CertificateLock.Lock()
	defer retVals.CertificateLock.Unlock()

	// ///// UNION (PORT PROTOCOL SERVICE) TUPLE INTO SET FOR DESTINATION IN CERTIFICATE DATA /////
	// Check if invalid cert record was written before the uconns
	// record, we'll need to update it with the tuples.
	if _, ok := retVals.CertificateMap[dstKey]; ok {
		// add tuple to invlaid cert list
		retVals.CertificateMap[dstKey].Tuples.Insert(tuple)
	}
}
