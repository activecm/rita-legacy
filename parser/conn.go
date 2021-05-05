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

	ts := parseConn.TimeStamp
	origIPBytes := parseConn.OrigIPBytes
	respIPBytes := parseConn.RespIPBytes
	duration := parseConn.Duration
	duration = math.Ceil((duration)*10000) / 10000
	bytes := int64(origIPBytes + respIPBytes)
	protocol := parseConn.Proto
	service := parseConn.Service
	dstPort := parseConn.DestinationPort
	var tuple string
	if service == "" {
		tuple = strconv.Itoa(dstPort) + ":" + protocol + ":-"
	} else {
		tuple = strconv.Itoa(dstPort) + ":" + protocol + ":" + service
	}

	// ///// CREATE HOST ENTRIES /////
	retVals.HostLock.Lock()
	// Check if the map value is set
	if _, ok := retVals.HostMap[srcKey]; !ok {
		// create new host record with src and dst
		retVals.HostMap[srcKey] = &host.Input{
			Host:    srcUniqIP,
			IsLocal: filter.checkIfInternal(srcIP),
			IP4:     util.IsIPv4(src),
			IP4Bin:  util.IPv4ToBinary(srcIP),
		}
	}
	retVals.HostLock.Unlock()

	retVals.HostLock.Lock()
	// Check if the map value is set
	if _, ok := retVals.HostMap[dstKey]; !ok {
		// create new host record with src and dst
		retVals.HostMap[dstKey] = &host.Input{
			Host:    dstUniqIP,
			IsLocal: filter.checkIfInternal(dstIP),
			IP4:     util.IsIPv4(dst),
			IP4Bin:  util.IPv4ToBinary(dstIP),
		}
	}
	retVals.HostLock.Unlock()

	// ///// CREATE UNIQUE CONNECTION ENTRY /////

	retVals.UniqueConnLock.Lock()
	// Check if the map value is set
	var uconnExists bool
	if _, uconnExists = retVals.UniqueConnMap[srcDstKey]; !uconnExists {
		// create new uconn record with src and dst
		// Set IsLocalSrc and IsLocalDst fields based on InternalSubnets setting
		// we only need to do this once if the uconn record does not exist
		retVals.UniqueConnMap[srcDstKey] = &uconn.Input{
			Hosts:      srcDstPair,
			IsLocalSrc: filter.checkIfInternal(srcIP),
			IsLocalDst: filter.checkIfInternal(dstIP),
		}

		// ///// INCREMENT SOURCE / DESTINATION COUNTERS FOR HOSTS /////
		// We only want to do this once for each unique connection entry
		retVals.HostLock.Lock()
		retVals.HostMap[srcKey].CountSrc++
		retVals.HostMap[dstKey].CountDst++
		retVals.HostLock.Unlock()
	}
	retVals.UniqueConnLock.Unlock()

	// ///// SET UNEXPECTED (PORT PROTOCOL SERVICE) FLAG /////
	// this is to keep track of how many times a host connected to
	// an unexpected port - proto - service Tuple
	// we only want to increment the count once per unique destination,
	// not once per connection, hence the flag and the check
	retVals.UniqueConnLock.Lock()
	setUPPSFlag := false
	if !retVals.UniqueConnMap[srcDstKey].UPPSFlag {
		for _, entry := range trustedAppReferenceList {
			if (protocol == entry.protocol) && (dstPort == entry.port) &&
				(service != entry.service) {
				retVals.UniqueConnMap[srcDstKey].UPPSFlag = true
				setUPPSFlag = true
				break
			}
		}
	}
	retVals.UniqueConnLock.Unlock()

	// ///// INCREMENT HOST UNEXPECTED (PORT PROTOCOL SERVICE) COUNTER /////
	// only do this once per unique connection
	if setUPPSFlag {
		retVals.HostLock.Lock()
		retVals.HostMap[srcKey].UntrustedAppConnCount++
		retVals.HostLock.Unlock()
	}

	// ///// UNION (PORT PROTOCOL SERVICE) TUPLE INTO SET FOR UNIQUE CONNECTION /////
	retVals.UniqueConnLock.Lock()
	if !util.StringInSlice(tuple, retVals.UniqueConnMap[srcDstKey].Tuples) {
		retVals.UniqueConnMap[srcDstKey].Tuples = append(
			retVals.UniqueConnMap[srcDstKey].Tuples, tuple,
		)
	}
	retVals.UniqueConnLock.Unlock()

	// ///// UNION (PORT PROTOCOL SERVICE) TUPLE INTO SET FOR DESTINATION IN CERTIFICATE DATA /////
	// Check if invalid cert record was written before the uconns
	// record, we'll need to update it with the tuples.
	retVals.CertificateLock.Lock()
	if _, ok := retVals.CertificateMap[dstKey]; ok {
		// add tuple to invlaid cert list
		if !util.StringInSlice(tuple, retVals.CertificateMap[dstKey].Tuples) {
			retVals.CertificateMap[dstKey].Tuples = append(
				retVals.CertificateMap[dstKey].Tuples, tuple,
			)
		}
	}
	retVals.CertificateLock.Unlock()

	// ///// INCREMENT THE CONNECTION COUNT FOR THE UNIQUE CONNECTION /////
	retVals.UniqueConnLock.Lock()
	retVals.UniqueConnMap[srcDstKey].ConnectionCount++
	retVals.UniqueConnLock.Unlock()

	// ///// INCREMENT THE CONNECTION COUNTS FOR THE HOSTS
	retVals.HostLock.Lock()
	retVals.HostMap[srcKey].ConnectionCount++
	retVals.HostMap[dstKey].ConnectionCount++
	retVals.HostLock.Unlock()

	// ///// UNION TIMESTAMP WITH UNIQUE CONNECTION TIMESTAMP SET /////
	retVals.UniqueConnLock.Lock()
	if !util.Int64InSlice(ts, retVals.UniqueConnMap[srcDstKey].TsList) {
		retVals.UniqueConnMap[srcDstKey].TsList = append(
			retVals.UniqueConnMap[srcDstKey].TsList, ts,
		)
	}
	retVals.UniqueConnLock.Unlock()

	// ///// APPEND IP BYTES TO UNIQUE CONNECTION BYTES LIST /////
	retVals.UniqueConnLock.Lock()
	retVals.UniqueConnMap[srcDstKey].OrigBytesList = append(
		retVals.UniqueConnMap[srcDstKey].OrigBytesList, origIPBytes,
	)
	retVals.UniqueConnLock.Unlock()

	// ///// ADD ORIG BYTES AND RESP BYTES TO UNIQUE CONNECTION TOTAL BYTES COUNTER /////
	// Calculate and store the total number of bytes exchanged by the uconn pair
	retVals.UniqueConnLock.Lock()
	retVals.UniqueConnMap[srcDstKey].TotalBytes += bytes
	retVals.UniqueConnLock.Unlock()

	// ///// ADD ORIG BYTES AND RESP BYTES TO EACH HOST'S TOTAL BYTES COUNTER /////
	retVals.HostLock.Lock()
	retVals.HostMap[srcKey].TotalBytes += bytes
	retVals.HostMap[dstKey].TotalBytes += bytes
	retVals.HostLock.Unlock()

	// ///// ADD CONNECTION DURATION TO UNIQUE CONNECTION'S TOTAL DURATION COUNTER /////
	retVals.UniqueConnLock.Lock()
	retVals.UniqueConnMap[srcDstKey].TotalDuration += duration
	retVals.UniqueConnLock.Unlock()

	// ///// ADD CONNECTION DURATION TO EACH HOST'S TOTAL DURATION COUNTER /////
	retVals.HostLock.Lock()
	retVals.HostMap[srcKey].TotalDuration += duration
	retVals.HostMap[dstKey].TotalDuration += duration
	retVals.HostLock.Unlock()

	// ///// DETERMINE THE LONGEST DURATION SEEN FOR THIS UNIQUE CONNECTION /////
	retVals.UniqueConnLock.Lock()
	// Replace existing duration if current duration is higher
	if duration > retVals.UniqueConnMap[srcDstKey].MaxDuration {
		retVals.UniqueConnMap[srcDstKey].MaxDuration = duration
	}
	retVals.UniqueConnLock.Unlock()

	// ///// DETERMINE THE LONGEST DURATION SEEN FOR THE SOURCE HOST /////
	retVals.HostLock.Lock()
	if duration > retVals.HostMap[srcKey].MaxDuration {
		retVals.HostMap[srcKey].MaxDuration = duration
	}
	retVals.HostLock.Unlock()

	// ///// DETERMINE THE LONGEST DURATION SEEN FOR THE DESTINATION HOST /////
	retVals.HostLock.Lock()
	if duration > retVals.HostMap[dstKey].MaxDuration {
		retVals.HostMap[dstKey].MaxDuration = duration
	}
	retVals.HostLock.Unlock()

}
