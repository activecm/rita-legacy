package parser

import (
	"net"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/certificate"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/pkg/useragent"
	"github.com/activecm/rita/util"
)

func parseSSLEntry(parseSSL *parsetypes.SSL, filter filter, retVals ParseResults) {
	ja3Hash := parseSSL.JA3
	src := parseSSL.Source
	dst := parseSSL.Destination
	host := parseSSL.ServerName
	certStatus := parseSSL.ValidationStatus

	srcIP := net.ParseIP(src)
	dstIP := net.ParseIP(dst)

	srcUniqIP := data.NewUniqueIP(srcIP, parseSSL.AgentUUID, parseSSL.AgentHostname)
	dstUniqIP := data.NewUniqueIP(dstIP, parseSSL.AgentUUID, parseSSL.AgentHostname)
	srcDstPair := data.NewUniqueIPPair(srcUniqIP, dstUniqIP)

	srcDstKey := srcDstPair.MapKey()
	dstKey := dstUniqIP.MapKey()

	if ja3Hash == "" {
		ja3Hash = "No JA3 hash generated"
	}

	certificateIsInvalid := certStatus != "ok" && certStatus != "-" && certStatus != "" && certStatus != " "

	// ///////////////////////// CREATE USERAGENT ENTRY /////////////////////////
	{
		retVals.UseragentLock.Lock()
		retVals.UseragentMap[ja3Hash] = &useragent.Input{
			Name: ja3Hash,
			JA3:  true,
		}
		retVals.UseragentLock.Unlock()
	}

	// ///////////////////////// USERAGENT UPDATES /////////////////////////
	{
		retVals.UseragentLock.Lock()
		// ///// INCREMENT USERAGENT COUNTER /////
		retVals.UseragentMap[ja3Hash].Seen++

		// ///// UNION SOURCE HOST INTO USERAGENT ORIGINATING HOSTS /////
		retVals.UseragentMap[ja3Hash].OrigIps.Insert(srcUniqIP)

		// ///// UNION DESTINATION HOSTNAME INTO USERAGENT DESTINATIONS /////
		if !util.StringInSlice(host, retVals.UseragentMap[ja3Hash].Requests) {
			retVals.UseragentMap[ja3Hash].Requests = append(retVals.UseragentMap[ja3Hash].Requests, host)
		}
		retVals.UseragentLock.Unlock()
	}

	// create uconn and cert records
	// Run conn pair through filter to filter out certain connections
	ignore := filter.filterConnPair(srcIP, dstIP)
	if ignore {
		return
	}

	// ///////////////////////// CREATE UNIQUE CONNECTION ENTRY /////////////////////////
	{
		retVals.UniqueConnLock.Lock()
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
		retVals.UniqueConnLock.Unlock()
	}

	// ///////////////////////// CREATE CERTIFICATE ENTRY /////////////////////////
	if certificateIsInvalid {
		// update relevant cert record
		retVals.CertificateLock.Lock()
		if _, ok := retVals.CertificateMap[dstKey]; !ok {
			// create new uconn record if it does not exist
			retVals.CertificateMap[dstKey] = &certificate.Input{
				Host: dstUniqIP,
			}
		}
		retVals.CertificateLock.Unlock()
	}

	// ///////////////////////// UNIQUE CONNECTION UPDATES /////////////////////////
	if certificateIsInvalid {
		retVals.UniqueConnLock.Lock()
		// ///// SET INVALID CERTIFICATE FLAG FOR UNIQUE CONNECTION /////
		retVals.UniqueConnMap[srcDstKey].InvalidCertFlag = true
		retVals.UniqueConnLock.Unlock()
	}

	// ///////////////////////// CERTIFICATE UPDATES /////////////////////////
	if certificateIsInvalid {
		retVals.CertificateLock.Lock()
		// ///// INCREMENT CONNECTION COUNTER FOR DESTINATION WITH INVALID CERTIFICATE /////
		retVals.CertificateMap[dstKey].Seen++

		// ///// UNION CERTIFICATE STATUS INTO SET OF CERTIFICATE STATUSES FOR DESTINATINO HOST /////
		if !util.StringInSlice(certStatus, retVals.CertificateMap[dstKey].InvalidCerts) {
			retVals.CertificateMap[dstKey].InvalidCerts = append(retVals.CertificateMap[dstKey].InvalidCerts, certStatus)
		}

		// ///// UNION SOURCE HOST INTO SET OF HOSTS WHICH FETCHED THE DESTINATION'S INVALID CERTIFICATE /////
		retVals.CertificateMap[dstKey].OrigIps.Insert(srcUniqIP)

		retVals.CertificateLock.Unlock()
	}

	// ///////////////////////// COPY UNIQUE CONNECTION DATA TO CERTIFICATE ENTRY /////////////////////////
	if certificateIsInvalid {
		// add uconn entry service tuples to certificate entry tuples
		retVals.UniqueConnLock.Lock()
		retVals.CertificateLock.Lock()

		// ///// UNION (PORT PROTOCOL SERVICE) TUPLES FROM UNIQUE CONNECTIONS ENTRY INTO CERTIFICATE ENTRY /////
		for _, tuple := range retVals.UniqueConnMap[srcDstKey].Tuples {
			if !util.StringInSlice(tuple, retVals.CertificateMap[dstKey].Tuples) {
				retVals.CertificateMap[dstKey].Tuples = append(
					retVals.CertificateMap[dstKey].Tuples, tuple,
				)
			}
		}

		retVals.CertificateLock.Unlock()
		retVals.UniqueConnLock.Unlock()
	}
}
