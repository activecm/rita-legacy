package parser

import (
	"net"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/certificate"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/sniconn"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/pkg/useragent"
	"github.com/activecm/rita/util"

	log "github.com/sirupsen/logrus"
)

func parseSSLEntry(parseSSL *parsetypes.SSL, filter filter, retVals ParseResults, logger *log.Logger) {
	src := parseSSL.Source
	dst := parseSSL.Destination
	certStatus := parseSSL.ValidationStatus

	// parse source and destination
	srcIP := net.ParseIP(src)
	dstIP := net.ParseIP(dst)

	// verify that both addresses were parsed successfully
	if (srcIP == nil) || (dstIP == nil) {
		logger.WithFields(log.Fields{
			"uid": parseSSL.UID,
			"src": parseSSL.Source,
			"dst": parseSSL.Destination,
		}).Error("Unable to parse valid ip address pair from ssl log entry, skipping entry.")
		return
	}

	// get fqdn
	fqdn := parseSSL.ServerName

	srcUniqIP := data.NewUniqueIP(srcIP, parseSSL.AgentUUID, parseSSL.AgentHostname)
	dstUniqIP := data.NewUniqueIP(dstIP, parseSSL.AgentUUID, parseSSL.AgentHostname)
	srcDstPair := data.NewUniqueIPPair(srcUniqIP, dstUniqIP)

	srcFQDNPair := data.NewUniqueSrcFQDNPair(srcUniqIP, fqdn)

	srcDstKey := srcDstPair.MapKey()
	srcKey := srcUniqIP.MapKey()
	dstKey := dstUniqIP.MapKey()

	srcFQDNKey := srcFQDNPair.MapKey()

	// create uconn and cert records
	// Run conn pair through filter to filter out certain connections
	ignore := filter.filterDomain(fqdn) || filter.filterConnPair(srcIP, dstIP)
	if ignore {
		return
	}

	updateUseragentsBySSL(srcUniqIP, parseSSL, retVals)

	certificateIsInvalid := certStatus != "ok" && certStatus != "-" && certStatus != "" && certStatus != " "

	newUniqueConnection := updateUniqueConnectionsBySSL(srcIP, dstIP, srcDstPair, srcDstKey, certificateIsInvalid, parseSSL, filter, retVals)

	updateTLSConnectionsBySSL(srcIP, dstUniqIP, srcFQDNPair, srcFQDNKey, certificateIsInvalid, parseSSL, filter, retVals)

	updateHostsBySSL(srcIP, dstIP, srcUniqIP, dstUniqIP, srcKey, dstKey, newUniqueConnection, filter, retVals)

	if certificateIsInvalid {
		updateCertificatesBySSL(srcUniqIP, dstUniqIP, dstKey, certStatus, retVals)
		// the unique connection record may have been created before the certificate record was seen
		copyServiceTuplesFromUconnToCerts(dstKey, srcDstKey, retVals)
	}
}

func updateUseragentsBySSL(srcUniqIP data.UniqueIP, parseSSL *parsetypes.SSL, retVals ParseResults) {

	retVals.UseragentLock.Lock()
	defer retVals.UseragentLock.Unlock()

	if parseSSL.JA3 == "" {
		parseSSL.JA3 = "No JA3 hash generated"
	}

	if _, ok := retVals.UseragentMap[parseSSL.JA3]; !ok {
		retVals.UseragentMap[parseSSL.JA3] = &useragent.Input{
			Name:     parseSSL.JA3,
			JA3:      true,
			OrigIps:  make(data.UniqueIPSet),
			Requests: make(data.StringSet),
		}
	}

	// ///// INCREMENT USERAGENT COUNTER /////
	retVals.UseragentMap[parseSSL.JA3].Seen++

	// ///// UNION SOURCE HOST INTO USERAGENT ORIGINATING HOSTS /////
	retVals.UseragentMap[parseSSL.JA3].OrigIps.Insert(srcUniqIP)

	// ///// UNION DESTINATION HOSTNAME INTO USERAGENT DESTINATIONS /////
	retVals.UseragentMap[parseSSL.JA3].Requests.Insert(parseSSL.ServerName)
}

func updateTLSConnectionsBySSL(srcIP net.IP, dstUniqIP data.UniqueIP, srcFQDNPair data.UniqueSrcFQDNPair, srcFQDNKey string,
	certificateIsInvalid bool, parseSSL *parsetypes.SSL, filter filter, retVals ParseResults) {

	if len(srcFQDNPair.FQDN) == 0 {
		return // don't record TLS SNI connections when the SNI is missing
	}

	retVals.TLSConnLock.Lock()
	defer retVals.TLSConnLock.Unlock()

	if _, ok := retVals.TLSConnMap[srcFQDNKey]; !ok {
		retVals.TLSConnMap[srcFQDNKey] = &sniconn.TLSInput{
			Hosts:           srcFQDNPair,
			IsLocalSrc:      filter.checkIfInternal(srcIP),
			Timestamps:      []int64{},
			RespondingIPs:   make(data.UniqueIPSet),
			RespondingPorts: make(data.IntSet),

			Subjects: make(data.StringSet),
			JA3s:     make(data.StringSet),
			JA3Ss:    make(data.StringSet),
		}
	}

	// ///// INCREMENT THE CONNECTION COUNT FOR THE TLS SNI CONNECTION /////
	retVals.TLSConnMap[srcFQDNKey].ConnectionCount++

	// ///// APPEND TIMESTAMP TO TLS TIMESTAMP LIST /////
	retVals.TLSConnMap[srcFQDNKey].Timestamps = append(
		retVals.TLSConnMap[srcFQDNKey].Timestamps, parseSSL.TimeStamp,
	)

	// ///// UNION DESTINATION HOST INTO TLS RESPONDING HOSTS /////
	retVals.TLSConnMap[srcFQDNKey].RespondingIPs.Insert(dstUniqIP)

	// ///// UNION DESTINATION PORT INTO TLS RESPONDING PORTS /////
	retVals.TLSConnMap[srcFQDNKey].RespondingPorts.Insert(parseSSL.DestinationPort)

	// ///// MARK RESPONDING CERTIFICATE AS INVALID /////
	retVals.TLSConnMap[srcFQDNKey].RespondingCertInvalid = certificateIsInvalid

	// ///// UNION RESPONDING CERTIFICATE SUBJECT INTO TLS SUBJECTS /////
	if len(parseSSL.Subject) > 0 {
		retVals.TLSConnMap[srcFQDNKey].Subjects.Insert(parseSSL.Subject)
	}

	// ///// UNION CLIENT JA3 HASH INTO TLS JA3 SET /////
	if len(parseSSL.JA3) > 0 {
		retVals.TLSConnMap[srcFQDNKey].JA3s.Insert(parseSSL.JA3)
	}

	// ///// UNION SERVER JA3S HASH INTO TLS JA3S SET /////
	if len(parseSSL.JA3S) > 0 {
		retVals.TLSConnMap[srcFQDNKey].JA3Ss.Insert(parseSSL.JA3S)
	}

	// ///// APPEND ZEEK RECORD UID INTO TLS UID SET /////
	// This allows us to link conn record information to this
	// ip -> fqdn record such as data sizes.
	if len(parseSSL.UID) > 0 {
		retVals.TLSConnMap[srcFQDNKey].ZeekUIDs = append(
			retVals.TLSConnMap[srcFQDNKey].ZeekUIDs,
			parseSSL.UID,
		)
	}
}

func updateUniqueConnectionsBySSL(srcIP, dstIP net.IP, srcDstPair data.UniqueIPPair, srcDstKey string,
	certificateIsInvalid bool, parseSSL *parsetypes.SSL, filter filter, retVals ParseResults) (newEntry bool) {

	retVals.UniqueConnLock.Lock()
	defer retVals.UniqueConnLock.Unlock()

	newEntry = false

	// Check if uconn map value is set, because this record could
	// come before a relevant uconns record (or may be the only source
	// for the uconns record)
	if _, ok := retVals.UniqueConnMap[srcDstKey]; !ok {
		newEntry = true

		// create new uconn record if it does not exist
		retVals.UniqueConnMap[srcDstKey] = &uconn.Input{
			Hosts:      srcDstPair,
			IsLocalSrc: filter.checkIfInternal(srcIP),
			IsLocalDst: filter.checkIfInternal(dstIP),
			Tuples:     make(data.StringSet),
		}
	}

	// ///// SET INVALID CERTIFICATE FLAG FOR UNIQUE CONNECTION /////
	if certificateIsInvalid {
		retVals.UniqueConnMap[srcDstKey].InvalidCertFlag = true
	}
	return
}

func updateHostsBySSL(srcIP, dstIP net.IP, srcUniqIP, dstUniqIP data.UniqueIP, srcKey, dstKey string,
	newUniqueConnection bool, filter filter, retVals ParseResults) {

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
}

func updateCertificatesBySSL(srcUniqIP data.UniqueIP, dstUniqIP data.UniqueIP, dstKey string,
	certStatus string, retVals ParseResults) {

	retVals.CertificateLock.Lock()
	defer retVals.CertificateLock.Unlock()

	if _, ok := retVals.CertificateMap[dstKey]; !ok {
		// create new uconn record if it does not exist
		retVals.CertificateMap[dstKey] = &certificate.Input{
			Host:         dstUniqIP,
			OrigIps:      make(data.UniqueIPSet),
			InvalidCerts: make(data.StringSet),
			Tuples:       make(data.StringSet),
		}
	}

	// ///// INCREMENT CONNECTION COUNTER FOR DESTINATION WITH INVALID CERTIFICATE /////
	retVals.CertificateMap[dstKey].Seen++

	// ///// UNION CERTIFICATE STATUS INTO SET OF CERTIFICATE STATUSES FOR DESTINATINO HOST /////
	retVals.CertificateMap[dstKey].InvalidCerts.Insert(certStatus)

	// ///// UNION SOURCE HOST INTO SET OF HOSTS WHICH FETCHED THE DESTINATION'S INVALID CERTIFICATE /////
	retVals.CertificateMap[dstKey].OrigIps.Insert(srcUniqIP)
}

func copyServiceTuplesFromUconnToCerts(dstKey, srcDstKey string, retVals ParseResults) {
	retVals.UniqueConnLock.Lock()
	retVals.CertificateLock.Lock()

	// ///// UNION (PORT PROTOCOL SERVICE) TUPLES FROM UNIQUE CONNECTIONS ENTRY INTO CERTIFICATE ENTRY /////
	for tuple := range retVals.UniqueConnMap[srcDstKey].Tuples {
		retVals.CertificateMap[dstKey].Tuples.Insert(tuple)
	}

	retVals.CertificateLock.Unlock()
	retVals.UniqueConnLock.Unlock()
}
