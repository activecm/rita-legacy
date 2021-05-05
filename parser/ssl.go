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

	// Safely store ja3 information

	// create useragent record if it doesn't exist
	retVals.UseragentLock.Lock()
	if _, ok := retVals.UseragentMap[ja3Hash]; !ok {
		retVals.UseragentMap[ja3Hash] = &useragent.Input{
			Name:     ja3Hash,
			Seen:     1,
			Requests: []string{host},
			JA3:      true,
		}
		retVals.UseragentMap[ja3Hash].OrigIps.Insert(srcUniqIP)
	} else {
		// increment times seen count
		retVals.UseragentMap[ja3Hash].Seen++

		// add src of ssl request to unique array
		retVals.UseragentMap[ja3Hash].OrigIps.Insert(srcUniqIP)

		// add request string to unique array
		if !util.StringInSlice(host, retVals.UseragentMap[ja3Hash].Requests) {
			retVals.UseragentMap[ja3Hash].Requests = append(retVals.UseragentMap[ja3Hash].Requests, host)
		}
	}
	retVals.UseragentLock.Unlock()

	// create uconn and cert records
	// Run conn pair through filter to filter out certain connections
	ignore := filter.filterConnPair(srcIP, dstIP)
	if ignore {
		return
	}

	// Check if uconn map value is set, because this record could
	// come before a relevant uconns record (or may be the only source
	// for the uconns record)
	retVals.UniqueConnLock.Lock()
	if _, ok := retVals.UniqueConnMap[srcDstKey]; !ok {
		// create new uconn record if it does not exist
		retVals.UniqueConnMap[srcDstKey] = &uconn.Input{
			Hosts:      srcDstPair,
			IsLocalSrc: filter.checkIfInternal(srcIP),
			IsLocalDst: filter.checkIfInternal(dstIP),
		}
	}
	retVals.UniqueConnLock.Unlock()

	//if there's any problem in the certificate, mark it invalid
	if certStatus != "ok" && certStatus != "-" && certStatus != "" && certStatus != " " {
		// mark as having invalid cert
		retVals.UniqueConnLock.Lock()
		retVals.UniqueConnMap[srcDstKey].InvalidCertFlag = true
		retVals.UniqueConnLock.Unlock()

		// update relevant cert record
		retVals.CertificateLock.Lock()
		if _, ok := retVals.CertificateMap[dstKey]; !ok {
			// create new uconn record if it does not exist
			retVals.CertificateMap[dstKey] = &certificate.Input{
				Host: dstUniqIP,
				Seen: 1,
			}
		} else {
			retVals.CertificateMap[dstKey].Seen++
		}
		retVals.CertificateLock.Unlock()

		// add uconn entry service tuples to certificate entry tuples
		retVals.UniqueConnLock.Lock()
		for _, tuple := range retVals.UniqueConnMap[srcDstKey].Tuples {
			retVals.CertificateLock.Lock()
			if !util.StringInSlice(tuple, retVals.CertificateMap[dstKey].Tuples) {
				retVals.CertificateMap[dstKey].Tuples = append(
					retVals.CertificateMap[dstKey].Tuples, tuple,
				)
			}
			retVals.CertificateLock.Unlock()
		}
		retVals.UniqueConnLock.Unlock()

		// mark as having invalid cert
		retVals.CertificateLock.Lock()
		if !util.StringInSlice(certStatus, retVals.CertificateMap[dstKey].InvalidCerts) {
			retVals.CertificateMap[dstKey].InvalidCerts = append(retVals.CertificateMap[dstKey].InvalidCerts, certStatus)
		}
		retVals.CertificateLock.Unlock()

		// add src of ssl request to unique array
		retVals.CertificateLock.Lock()
		retVals.CertificateMap[dstKey].OrigIps.Insert(srcUniqIP)
		retVals.CertificateLock.Unlock()
	}
}
