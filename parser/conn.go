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

func parseConnEntry(parseConn *parsetypes.Conn, filter filter, retVals ParseResults) {
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

	// disambiguate addresses which are not publicly routable
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

	newUniqueConnection, setUPPSFlag := updateUniqueConnectionsByConn(
		srcIP, dstIP, srcDstPair, srcDstKey, roundedDuration, twoWayIPBytes, tuple, parseConn, filter, retVals,
	)

	updateHostsByConn(
		srcIP, dstIP, srcUniqIP, dstUniqIP, srcKey, dstKey, newUniqueConnection, setUPPSFlag,
		roundedDuration, twoWayIPBytes, tuple, parseConn, filter, retVals,
	)

	updateCertificatesByConn(dstKey, tuple, retVals)
}

func updateUniqueConnectionsByConn(srcIP, dstIP net.IP, srcDstPair data.UniqueIPPair, srcDstKey string,
	roundedDuration float64, twoWayIPBytes int64, tuple string,
	parseConn *parsetypes.Conn, filter filter, retVals ParseResults) (newEntry bool, setUPPSFlag bool) {

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

	// ///// MARK UNIQUE CONNECTION AS CLOSED /////
	// If the ConnStateList map doesn't exist for this entry, create it.
	if retVals.UniqueConnMap[srcDstKey].ConnStateMap == nil {
		retVals.UniqueConnMap[srcDstKey].ConnStateMap = make(map[string]*uconn.ConnState)
	}
	// If an entry for this connection is present, it means it was
	// marked as open and we should now mark it as closed. No need
	// to update any other attributes as we are going to discard
	// them later anyways since the data will now go into the
	// dat section of the uconn entry
	if _, ok := retVals.UniqueConnMap[srcDstKey].ConnStateMap[parseConn.UID]; ok {
		retVals.UniqueConnMap[srcDstKey].ConnStateMap[parseConn.UID].Open = false
	}

	// ///// UNION (PORT PROTOCOL SERVICE) TUPLE INTO SET FOR UNIQUE CONNECTION /////
	retVals.UniqueConnMap[srcDstKey].Tuples.Insert(tuple)

	// ///// INCREMENT THE CONNECTION COUNT FOR THE UNIQUE CONNECTION /////
	retVals.UniqueConnMap[srcDstKey].ConnectionCount++

	// ///// UNION TIMESTAMP WITH UNIQUE CONNECTION TIMESTAMP SET /////
	if !util.Int64InSlice(parseConn.TimeStamp, retVals.UniqueConnMap[srcDstKey].TsList) {
		retVals.UniqueConnMap[srcDstKey].TsList = append(
			retVals.UniqueConnMap[srcDstKey].TsList, parseConn.TimeStamp,
		)
	}

	// ///// APPEND IP BYTES TO UNIQUE CONNECTION BYTES LIST /////
	retVals.UniqueConnMap[srcDstKey].OrigBytesList = append(
		retVals.UniqueConnMap[srcDstKey].OrigBytesList, parseConn.OrigIPBytes,
	)

	// ///// ADD ORIG BYTES AND RESP BYTES TO UNIQUE CONNECTION TOTAL BYTES COUNTER /////
	// Calculate and store the total number of bytes exchanged by the uconn pair
	retVals.UniqueConnMap[srcDstKey].TotalBytes += twoWayIPBytes

	// ///// ADD CONNECTION DURATION TO UNIQUE CONNECTION'S TOTAL DURATION COUNTER /////
	retVals.UniqueConnMap[srcDstKey].TotalDuration += roundedDuration

	// ///// DETERMINE THE LONGEST DURATION SEEN FOR THIS UNIQUE CONNECTION /////
	// Replace existing duration if current duration is higher
	if roundedDuration > retVals.UniqueConnMap[srcDstKey].MaxDuration {
		retVals.UniqueConnMap[srcDstKey].MaxDuration = roundedDuration
	}

	return
}

func updateHostsByConn(srcIP, dstIP net.IP, srcUniqIP, dstUniqIP data.UniqueIP, srcKey, dstKey string,
	newUniqueConnection, setUPPSFlag bool, roundedDuration float64, twoWayIPBytes int64, tuple string,
	parseConn *parsetypes.Conn, filter filter, retVals ParseResults) {

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
	if newUniqueConnection {
		retVals.HostMap[srcKey].CountSrc++
		retVals.HostMap[dstKey].CountDst++
	}

	// ///// INCREMENT THE CONNECTION COUNTS FOR THE HOSTS
	retVals.HostMap[srcKey].ConnectionCount++
	retVals.HostMap[dstKey].ConnectionCount++

	// ///// INCREMENT HOST UNEXPECTED (PORT PROTOCOL SERVICE) COUNTER /////
	// only do this once per flagged unique connection
	if setUPPSFlag {
		retVals.HostMap[srcKey].UntrustedAppConnCount++
	}

	// ///// ADD ORIG BYTES AND RESP BYTES TO EACH HOST'S TOTAL BYTES COUNTER /////
	// Not sure that this is used anywhere?
	retVals.HostMap[srcKey].TotalBytes += twoWayIPBytes
	retVals.HostMap[dstKey].TotalBytes += twoWayIPBytes

	// ///// ADD CONNECTION DURATION TO EACH HOST'S TOTAL DURATION COUNTER /////
	// Not sure that this is used anywhere?
	retVals.HostMap[srcKey].TotalDuration += roundedDuration
	retVals.HostMap[dstKey].TotalDuration += roundedDuration

	// ///// DETERMINE THE LONGEST DURATION SEEN FOR THE SOURCE HOST /////
	if roundedDuration > retVals.HostMap[srcKey].MaxDuration {
		retVals.HostMap[srcKey].MaxDuration = roundedDuration
	}

	// ///// DETERMINE THE LONGEST DURATION SEEN FOR THE DESTINATION HOST /////
	if roundedDuration > retVals.HostMap[dstKey].MaxDuration {
		retVals.HostMap[dstKey].MaxDuration = roundedDuration
	}
}

func updateCertificatesByConn(dstKey string, tuple string, retVals ParseResults) {

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
