package parser

import (
	"net"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/certificate"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/pkg/useragent"
)

func parseSSLEntry(parseSSL *parsetypes.SSL, filter filter, retVals ParseResults) {
	src := parseSSL.Source
	dst := parseSSL.Destination
	certStatus := parseSSL.ValidationStatus

	srcIP := net.ParseIP(src)
	dstIP := net.ParseIP(dst)

	srcUniqIP := data.NewUniqueIP(srcIP, parseSSL.AgentUUID, parseSSL.AgentHostname)
	dstUniqIP := data.NewUniqueIP(dstIP, parseSSL.AgentUUID, parseSSL.AgentHostname)
	srcDstPair := data.NewUniqueIPPair(srcUniqIP, dstUniqIP)

	srcDstKey := srcDstPair.MapKey()
	dstKey := dstUniqIP.MapKey()

	updateUseragentsBySSL(srcUniqIP, parseSSL, retVals)

	// create uconn and cert records
	// Run conn pair through filter to filter out certain connections
	ignore := filter.filterConnPair(srcIP, dstIP)
	if ignore {
		return
	}

	certificateIsInvalid := certStatus != "ok" && certStatus != "-" && certStatus != "" && certStatus != " "

	updateUniqueConnectionsBySSL(srcIP, dstIP, srcDstPair, srcDstKey, certificateIsInvalid, parseSSL, filter, retVals)

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
			Name: parseSSL.JA3,
			JA3:  true,
		}
	}

	// ///// INCREMENT USERAGENT COUNTER /////
	retVals.UseragentMap[parseSSL.JA3].Seen++

	// ///// UNION SOURCE HOST INTO USERAGENT ORIGINATING HOSTS /////
	retVals.UseragentMap[parseSSL.JA3].OrigIps.Insert(srcUniqIP)

	// ///// UNION DESTINATION HOSTNAME INTO USERAGENT DESTINATIONS /////
	retVals.UseragentMap[parseSSL.JA3].Requests.Insert(parseSSL.ServerName)
}

func updateUniqueConnectionsBySSL(srcIP, dstIP net.IP, srcDstPair data.UniqueIPPair, srcDstKey string,
	certificateIsInvalid bool, parseSSL *parsetypes.SSL, filter filter, retVals ParseResults) {

	retVals.UniqueConnLock.Lock()
	defer retVals.UniqueConnLock.Unlock()

	// Check if uconn map value is set, because this record could
	// come before a relevant uconns record (or may be the only source
	// for the uconns record)
	if _, ok := retVals.UniqueConnMap[srcDstKey]; !ok {
		// create new uconn record if it does not exist
		retVals.UniqueConnMap[srcDstKey] = &uconn.Input{
			Hosts:      srcDstPair,
			IsLocalSrc: filter.checkIfInternal(srcIP),
			IsLocalDst: filter.checkIfInternal(dstIP),
		}
	}

	// ///// SET INVALID CERTIFICATE FLAG FOR UNIQUE CONNECTION /////
	if certificateIsInvalid {
		retVals.UniqueConnMap[srcDstKey].InvalidCertFlag = true
	}
}

func updateCertificatesBySSL(srcUniqIP data.UniqueIP, dstUniqIP data.UniqueIP, dstKey string,
	certStatus string, retVals ParseResults) {

	retVals.CertificateLock.Lock()
	defer retVals.CertificateLock.Unlock()

	if _, ok := retVals.CertificateMap[dstKey]; !ok {
		// create new uconn record if it does not exist
		retVals.CertificateMap[dstKey] = &certificate.Input{
			Host: dstUniqIP,
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
