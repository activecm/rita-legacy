package parser

import (
	"fmt"
	"math"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/parser/files"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/beacon"
	"github.com/activecm/rita/pkg/beaconfqdn"
	"github.com/activecm/rita/pkg/beaconproxy"
	"github.com/activecm/rita/pkg/blacklist"
	"github.com/activecm/rita/pkg/certificate"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/explodeddns"

	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/pkg/remover"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/pkg/useragent"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	//FSImporter provides the ability to import bro files from the file system
	FSImporter struct {
		filter

		log      *log.Logger
		config   *config.Config
		database *database.DB
		metaDB   *database.MetaDB

		batchSizeBytes int64
	}

	trustedAppTiplet struct {
		protocol string
		port     int
		service  string
	}
)

//NewFSImporter creates a new file system importer
func NewFSImporter(res *resources.Resources) *FSImporter {
	return &FSImporter{
		filter:         newFilter(res.Config),
		log:            res.Log,
		config:         res.Config,
		database:       res.DB,
		metaDB:         res.MetaDB,
		batchSizeBytes: 2 * (2 << 30), // 2 gigabytes (used to not run out of memory while importing)
	}
}

var trustedAppReferenceList = [...]trustedAppTiplet{
	{"tcp", 80, "http"},
	{"tcp", 443, "ssl"},
}

//GetInternalSubnets returns the internal subnets from the config file
func (fs *FSImporter) GetInternalSubnets() []*net.IPNet {
	return fs.internal
}

//CollectFileDetails reads and hashes the files
func (fs *FSImporter) CollectFileDetails(importFiles []string, threads int) []*files.IndexedFile {
	// find all of the potential bro log paths
	logFiles := files.GatherLogFiles(importFiles, fs.log)

	// hash the files and get their stats
	return files.IndexFiles(
		logFiles, threads, fs.database.GetSelectedDB(), fs.config.S.Rolling.CurrentChunk, fs.log, fs.config,
	)
}

//Run starts the importing
func (fs *FSImporter) Run(indexedFiles []*files.IndexedFile, threads int) {
	start := time.Now()

	fmt.Println("\t[-] Verifying log files have not been previously parsed into the target dataset ... ")
	// check list of files against metadatabase records to ensure that the a file
	// won't be imported into the same database twice.
	indexedFiles = fs.metaDB.FilterOutPreviouslyIndexedFiles(indexedFiles, fs.database.GetSelectedDB())

	// if all files were removed because they've already been imported, handle error
	if !(len(indexedFiles) > 0) {
		if fs.config.S.Rolling.Rolling {
			fmt.Println("\t[!] All files pertaining to the current chunk entry have already been parsed into database: ", fs.database.GetSelectedDB())
		} else {
			fmt.Println("\t[!] All files in this directory have already been parsed into database: ", fs.database.GetSelectedDB())
		}
		return
	}

	// Add new metadatabase record for db if doesn't already exist
	dbExists, err := fs.metaDB.DBExists(fs.database.GetSelectedDB())
	if err != nil {
		fs.log.WithFields(log.Fields{
			"err":      err,
			"database": fs.database.GetSelectedDB(),
		}).Error("Could not check if metadatabase record exists for target database")
		fmt.Printf("\t[!] %v", err.Error())
	}

	if !dbExists {
		err := fs.metaDB.AddNewDB(fs.database.GetSelectedDB(), fs.config.S.Rolling.CurrentChunk, fs.config.S.Rolling.TotalChunks)
		if err != nil {
			fs.log.WithFields(log.Fields{
				"err":      err,
				"database": fs.database.GetSelectedDB(),
			}).Error("Could not add metadatabase record for new database")
			fmt.Printf("\t[!] %v", err.Error())
		}
	}

	if fs.config.S.Rolling.Rolling {
		err := fs.metaDB.SetRollingSettings(fs.database.GetSelectedDB(), fs.config.S.Rolling.CurrentChunk, fs.config.S.Rolling.TotalChunks)
		if err != nil {
			fs.log.WithFields(log.Fields{
				"err":      err,
				"database": fs.database.GetSelectedDB(),
			}).Error("Could not update rolling database settings for database")
			fmt.Printf("\t[!] %v", err.Error())
		}

		chunkSet, err := fs.metaDB.IsChunkSet(fs.config.S.Rolling.CurrentChunk, fs.database.GetSelectedDB())
		if err != nil {
			fmt.Println("\t[!] Could not find CID List entry in metadatabase")
			return
		}

		if chunkSet {
			fmt.Println("\t[-] Removing outdated data from rolling dataset ... ")
			err := fs.removeAnalysisChunk(fs.config.S.Rolling.CurrentChunk)
			if err != nil {
				fmt.Println("\t[!] Failed to remove outdata data from rolling dataset")
				return
			}
		}
	}

	// create blacklisted reference Collection if blacklisted module is enabled
	if fs.config.S.Blacklisted.Enabled {
		blacklist.BuildBlacklistedCollections(fs.database, fs.config, fs.log)
	}

	// batch up the indexed files so as not to read too much in at one time
	batchedIndexedFiles := batchFilesBySize(indexedFiles, fs.batchSizeBytes)

	for i, indexedFileBatch := range batchedIndexedFiles {
		fmt.Printf("\t[-] Processing batch %d of %d\n", i+1, len(batchedIndexedFiles))

		// parse in those files!
		uconnMap, hostMap, explodeddnsMap, hostnameMap, proxyHostnameMap, useragentMap, certMap := fs.parseFiles(indexedFileBatch, threads, fs.log)

		// Set chunk before we continue so if process dies, we still verify with a delete if
		// any data was written out.
		fs.metaDB.SetChunk(fs.config.S.Rolling.CurrentChunk, fs.database.GetSelectedDB(), true)

		// build Hosts table.
		fs.buildHosts(hostMap)

		// build Uconns table. Must go before beacons.
		fs.buildUconns(uconnMap)

		// update ts range for dataset (needs to be run before beacons)
		minTimestamp, maxTimestamp := fs.updateTimestampRange()

		// build or update the exploded DNS table. Must go before hostnames
		fs.buildExplodedDNS(explodeddnsMap)

		// build or update the exploded DNS table
		fs.buildHostnames(hostnameMap)

		// build or update Beacons table
		fs.buildBeacons(uconnMap, minTimestamp, maxTimestamp)

		// build or update the FQDN Beacons Table
		fs.buildFQDNBeacons(hostnameMap, minTimestamp, maxTimestamp)

		// build or update the Proxy Beacons Table
		fs.buildProxyBeacons(proxyHostnameMap, minTimestamp, maxTimestamp)

		// build or update UserAgent table
		fs.buildUserAgent(useragentMap)

		// build or update Certificate table
		fs.buildCertificates(certMap)

		// update blacklisted peers in hosts collection
		fs.markBlacklistedPeers(hostMap)

		// record file+database name hash in metadabase to prevent duplicate content
		fmt.Println("\t[-] Indexing log entries ... ")
		err := fs.metaDB.AddNewFilesToIndex(indexedFileBatch)
		if err != nil {
			fs.log.Error("Could not update the list of parsed files")
		}
	}

	// mark results as imported and analyzed
	fmt.Println("\t[-] Updating metadatabase ... ")
	fs.metaDB.MarkDBAnalyzed(fs.database.GetSelectedDB(), true)

	progTime := time.Now()
	fs.log.WithFields(
		log.Fields{
			"current_time": progTime.Format(util.TimeFormat),
			"total_time":   progTime.Sub(start).String(),
		},
	).Info("Finished upload. Starting indexing")

	progTime = time.Now()
	fs.log.WithFields(
		log.Fields{
			"current_time": progTime.Format(util.TimeFormat),
			"total_time":   progTime.Sub(start).String(),
		},
	).Info("Finished importing log files")

	fmt.Println("\t[-] Done!")
}

// batchFilesBySize takes in an slice of indexedFiles and splits the array into
// subgroups of indexedFiles such that each group has a total size in bytes less than size
func batchFilesBySize(indexedFiles []*files.IndexedFile, size int64) [][]*files.IndexedFile {
	// sort the indexed files so we process them in order
	sort.Slice(indexedFiles, func(i, j int) bool {
		return indexedFiles[i].Path < indexedFiles[j].Path
	})

	//group by target collection
	fileTypeMap := make(map[string][]*files.IndexedFile)
	for _, file := range indexedFiles {
		if _, ok := fileTypeMap[file.TargetCollection]; !ok {
			fileTypeMap[file.TargetCollection] = make([]*files.IndexedFile, 0)
		}
		fileTypeMap[file.TargetCollection] = append(fileTypeMap[file.TargetCollection], file)
	}

	// Take n files in each target collection group until the size limit is exceeded, then start a new batch
	batches := make([][]*files.IndexedFile, 0)
	currBatch := make([]*files.IndexedFile, 0)
	currAggBytes := int64(0)
	iterators := make(map[string]int)
	for fileType := range fileTypeMap {
		iterators[fileType] = 0
	}

	for len(iterators) != 0 { // while there is data to iterate through

		maybeBatch := make([]*files.IndexedFile, 0)
		maybeAggBytes := int64(0)
		for fileType := range fileTypeMap { // grab a file for each target collection.
			if _, ok := iterators[fileType]; ok { // if we haven't ran through all the files for the target collection
				// append the file and aggregate the bytes
				maybeBatch = append(maybeBatch, fileTypeMap[fileType][iterators[fileType]])
				maybeAggBytes += fileTypeMap[fileType][iterators[fileType]].Length
				iterators[fileType]++

				// if we've exhausted the files for the target collection, prevent accessing the map for that target collection
				if iterators[fileType] == len(fileTypeMap[fileType]) {
					delete(iterators, fileType)
				}
			}
		}

		// split off the current batch if adding the next set of files would exceed the target size
		// Guarding against the len(currBatch) == 0 case prevents us from failing when we cannot make
		// small enough batch sizes. We just try our best and process 1 file for each target collection at a time.
		if len(currBatch) != 0 && currAggBytes+maybeAggBytes >= size {
			batches = append(batches, currBatch)
			currBatch = make([]*files.IndexedFile, 0)
			currAggBytes = 0
		}

		currBatch = append(currBatch, maybeBatch...)
		currAggBytes += maybeAggBytes
	}
	batches = append(batches, currBatch) // add the last batch
	return batches
}

//parseFiles takes in a list of indexed bro files, the number of
//threads to use to parse the files, whether or not to sort data by date,
//a MongoDB datastore object to store the bro data in, and a logger to report
//errors and parses the bro files line by line into the database.
func (fs *FSImporter) parseFiles(indexedFiles []*files.IndexedFile, parsingThreads int, logger *log.Logger) (
	map[string]*uconn.Input, map[string]*host.Input, map[string]int, map[string]*hostname.Input, map[string]*beaconproxy.Input, map[string]*useragent.Input, map[string]*certificate.Input) {

	fmt.Println("\t[-] Parsing logs to: " + fs.database.GetSelectedDB() + " ... ")

	// create log parsing maps
	explodeddnsMap := make(map[string]int)

	hostnameMap := make(map[string]*hostname.Input)

	proxyHostnameMap := make(map[string]*beaconproxy.Input)

	useragentMap := make(map[string]*useragent.Input)

	certMap := make(map[string]*certificate.Input)

	// Counts the number of uconns per source-destination pair
	uconnMap := make(map[string]*uconn.Input)

	hostMap := make(map[string]*host.Input)

	//set up parallel parsing
	n := len(indexedFiles)
	parsingWG := new(sync.WaitGroup)

	// Creates a mutex for locking map keys during read-write operations
	var mutex = &sync.Mutex{}

	for i := 0; i < parsingThreads; i++ {
		parsingWG.Add(1)

		go func(indexedFiles []*files.IndexedFile, logger *log.Logger,
			wg *sync.WaitGroup, start int, jump int, length int) {
			//comb over array
			for j := start; j < length; j += jump {

				// open the file
				fileHandle, err := os.Open(indexedFiles[j].Path)
				if err != nil {
					logger.WithFields(log.Fields{
						"file":  indexedFiles[j].Path,
						"error": err.Error(),
					}).Error("Could not open file for parsing")
				}

				// read the file
				fileScanner, err := files.GetFileScanner(fileHandle)
				if err != nil {
					logger.WithFields(log.Fields{
						"file":  indexedFiles[j].Path,
						"error": err.Error(),
					}).Error("Could not read from the file")
				}
				fmt.Println("\t[-] Parsing " + indexedFiles[j].Path + " -> " + indexedFiles[j].TargetDatabase)

				// This loops through every line of the file
				for fileScanner.Scan() {
					// go to next line if there was an issue
					if fileScanner.Err() != nil {
						break
					}

					//parse the line
					var datum parsetypes.BroData
					if indexedFiles[j].IsJSON() {
						datum = files.ParseJSONLine(fileScanner.Bytes(), indexedFiles[j].GetBroDataFactory(), logger)
					} else {
						datum = files.ParseTSVLine(
							fileScanner.Text(), indexedFiles[j].GetHeader(), indexedFiles[j].GetFieldMap(),
							indexedFiles[j].GetBroDataFactory(), logger,
						)
					}

					if datum == nil {
						continue
					}

					if datum != nil {
						//figure out which collection (dns, http, or conn/conn_long) this line is heading for
						targetCollection := indexedFiles[j].TargetCollection

						switch targetCollection {

						/// *************************************************************///
						///                           CONNS                              ///
						/// *************************************************************///
						case fs.config.T.Structure.ConnTable:

							parseConn, ok := datum.(*parsetypes.Conn)
							if !ok {
								continue
							}

							// get source destination pair for connection record
							src := parseConn.Source
							dst := parseConn.Destination

							// parse addresses into binary format
							srcIP := net.ParseIP(src)
							dstIP := net.ParseIP(dst)

							// disambiguate addresses which are not publicly routable
							srcUniqIP := data.NewUniqueIP(srcIP, parseConn.AgentUUID, parseConn.AgentHostname)
							dstUniqIP := data.NewUniqueIP(dstIP, parseConn.AgentUUID, parseConn.AgentHostname)
							srcDstPair := data.NewUniqueIPPair(srcUniqIP, dstUniqIP)

							// get aggregation keys for ip addresses and connection pair
							srcKey := srcUniqIP.MapKey()
							dstKey := dstUniqIP.MapKey()
							srcDstKey := srcDstPair.MapKey()

							// Run conn pair through filter to filter out certain connections
							ignore := fs.filterConnPair(srcIP, dstIP)

							// If connection pair is not subject to filtering, process
							if !ignore {
								ts := parseConn.TimeStamp
								origIPBytes := parseConn.OrigIPBytes
								respIPBytes := parseConn.RespIPBytes
								duration := parseConn.Duration
								duration = math.Ceil((duration)*10000) / 10000
								bytes := int64(origIPBytes + respIPBytes)
								protocol := parseConn.Proto
								service := parseConn.Service
								dstPort := parseConn.DestinationPort
								uid := parseConn.UID

								var tuple string
								if service == "" {
									tuple = strconv.Itoa(dstPort) + ":" + protocol + ":-"
								} else {
									tuple = strconv.Itoa(dstPort) + ":" + protocol + ":" + service
								}

								// Safely store the number of conns for this uconn
								mutex.Lock()

								// Check if the map value is set
								if _, ok := hostMap[srcKey]; !ok {
									// create new host record with src and dst
									hostMap[srcKey] = &host.Input{
										Host:    srcUniqIP,
										IsLocal: fs.checkIfInternal(srcIP),
										IP4:     util.IsIPv4(src),
										IP4Bin:  util.IPv4ToBinary(srcIP),
									}
								}

								// Check if the map value is set
								if _, ok := hostMap[dstKey]; !ok {
									// create new host record with src and dst
									hostMap[dstKey] = &host.Input{
										Host:    dstUniqIP,
										IsLocal: fs.checkIfInternal(dstIP),
										IP4:     util.IsIPv4(dst),
										IP4Bin:  util.IPv4ToBinary(dstIP),
									}
								}

								// Check if the map value is set
								if _, ok := uconnMap[srcDstKey]; !ok {
									// create new uconn record with src and dst
									// Set IsLocalSrc and IsLocalDst fields based on InternalSubnets setting
									// we only need to do this once if the uconn record does not exist
									uconnMap[srcDstKey] = &uconn.Input{
										Hosts:      srcDstPair,
										IsLocalSrc: fs.checkIfInternal(srcIP),
										IsLocalDst: fs.checkIfInternal(dstIP),
									}

									hostMap[srcKey].CountSrc++
									hostMap[dstKey].CountDst++
								}

								// If the ConnStateList map doesn't exist for this entry, create it.
								if uconnMap[srcDstKey].ConnStateMap == nil {
									uconnMap[srcDstKey].ConnStateMap = make(map[string]*uconn.ConnState)
								}

								// If an entry for this connection is present, it means it was
								// marked as open and we should now mark it as closed. No need
								// to update any other attributes as we are going to discard
								// them later anyways since the data will now go into the
								// dat section of the uconn entry
								if _, ok := uconnMap[srcDstKey].ConnStateMap[uid]; ok {

									uconnMap[srcDstKey].ConnStateMap[uid].Open = false
								}

								// this is to keep track of how many times a host connected to
								// an unexpected port - proto - service Tuple
								// we only want to increment the count once per unique destination,
								// not once per connection, hence the flag and the check
								if !uconnMap[srcDstKey].UPPSFlag {
									for _, entry := range trustedAppReferenceList {
										if (protocol == entry.protocol) && (dstPort == entry.port) {
											if service != entry.service {
												uconnMap[srcDstKey].UPPSFlag = true

												hostMap[srcKey].UntrustedAppConnCount++
											}
										}
									}
								}

								// increment unique dst port: proto : service tuple list for host.
								if !stringInSlice(tuple, uconnMap[srcDstKey].Tuples) {
									uconnMap[srcDstKey].Tuples = append(uconnMap[srcDstKey].Tuples, tuple)
								}

								// Check if invalid cert record was written before the uconns
								// record, we'll need to update it with the tuples.
								if _, ok := certMap[dstKey]; ok {
									// add tuple to invlaid cert list
									if !stringInSlice(tuple, certMap[dstKey].Tuples) {
										certMap[dstKey].Tuples = append(certMap[dstKey].Tuples, tuple)
									}
								}

								// Replace existing duration if current duration is higher.
								if duration > uconnMap[srcDstKey].MaxDuration {
									uconnMap[srcDstKey].MaxDuration = duration
								}

								if duration > hostMap[srcKey].MaxDuration {
									hostMap[srcKey].MaxDuration = duration
								}
								if duration > hostMap[dstKey].MaxDuration {
									hostMap[dstKey].MaxDuration = duration
								}

								// Increment the connection count for the src-dst pair
								uconnMap[srcDstKey].ConnectionCount++

								// Not sure if this is used anywhere?
								hostMap[srcKey].ConnectionCount++
								hostMap[dstKey].ConnectionCount++

								// Only append unique timestamps to tslist
								if !int64InSlice(ts, uconnMap[srcDstKey].TsList) {
									uconnMap[srcDstKey].TsList = append(uconnMap[srcDstKey].TsList, ts)
								}

								// Append all origIPBytes to origBytesList
								uconnMap[srcDstKey].OrigBytesList = append(uconnMap[srcDstKey].OrigBytesList, origIPBytes)

								// Calculate and store the total number of bytes exchanged by the uconn pair
								uconnMap[srcDstKey].TotalBytes += bytes

								// Not sure that this is used anywhere?
								hostMap[srcKey].TotalBytes += bytes
								hostMap[dstKey].TotalBytes += bytes

								// Calculate and store the total duration
								uconnMap[srcDstKey].TotalDuration += duration

								// Not sure that this is used anywhere?
								hostMap[srcKey].TotalDuration += duration
								hostMap[dstKey].TotalDuration += duration

								mutex.Unlock()

							}

						/// *************************************************************///
						///                             DNS                              ///
						/// *************************************************************///
						case fs.config.T.Structure.DNSTable:
							parseDNS, ok := datum.(*parsetypes.DNS)
							if !ok {
								continue
							}

							domain := parseDNS.Query
							queryTypeName := parseDNS.QTypeName

							// extract and store the dns client ip address
							src := parseDNS.Source
							srcIP := net.ParseIP(src)

							// Run domain through filter to filter out certain domains
							ignore := (fs.filterDomain(domain) || fs.filterSingleIP(srcIP))

							// If domain is not subject to filtering, process
							if !ignore {

								// Safely store the number of conns for this uconn
								mutex.Lock()

								// increment domain map count for exploded dns
								explodeddnsMap[domain]++

								// initialize the hostname input objects for new hostnames
								if _, ok := hostnameMap[domain]; !ok {
									hostnameMap[domain] = &hostname.Input{
										Host: domain,
									}
								}

								srcUniqIP := data.NewUniqueIP(srcIP, parseDNS.AgentUUID, parseDNS.AgentHostname)
								srcKey := srcUniqIP.MapKey()

								hostnameMap[domain].ClientIPs.Insert(srcUniqIP)

								if queryTypeName == "A" {
									answers := parseDNS.Answers
									for _, answer := range answers {
										answerIP := net.ParseIP(answer)
										// Check if answer is an IP address and store it if it is
										if answerIP != nil {
											answerUniqIP := data.NewUniqueIP(answerIP, parseDNS.AgentUUID, parseDNS.AgentHostname)
											hostnameMap[domain].ResolvedIPs.Insert(answerUniqIP)
										}
									}
								}

								// We don't filter out the src ips like we do with the conn
								// section since a c2 channel running over dns could have an
								// internal ip to internal ip connection and not having that ip
								// in the host table is limiting

								// in some of these strings, the empty space will get counted as a domain,
								// don't add host or increment dns query count if queried domain
								// is blank or ends in 'in-addr.arpa'
								if (domain != "") && (!strings.HasSuffix(domain, "in-addr.arpa")) {
									// Check if host map value is set, because this record could
									// come before a relevant conns record
									if _, ok := hostMap[srcKey]; !ok {
										// create new uconn record with src and dst
										// Set IsLocalSrc and IsLocalDst fields based on InternalSubnets setting
										// we only need to do this once if the uconn record does not exist
										hostMap[srcKey] = &host.Input{
											Host:    srcUniqIP,
											IsLocal: fs.checkIfInternal(srcIP),
											IP4:     util.IsIPv4(src),
											IP4Bin:  util.IPv4ToBinary(srcIP),
										}
									}

									// if there are no entries in the dnsquerycount map for this
									// srcKey, initialize map
									if hostMap[srcKey].DNSQueryCount == nil {
										hostMap[srcKey].DNSQueryCount = make(map[string]int64)
									}

									// increment the dns query count for this domain
									hostMap[srcKey].DNSQueryCount[domain]++
								}

								mutex.Unlock()
							}

						/// *************************************************************///
						///                             HTTP                             ///
						/// *************************************************************///
						case fs.config.T.Structure.HTTPTable:
							parseHTTP, ok := datum.(*parsetypes.HTTP)
							if !ok {
								continue
							}

							// get source destination pair for connection record
							src := parseHTTP.Source
							dst := parseHTTP.Destination

							// parse addresses into binary format
							srcIP := net.ParseIP(src)
							dstIP := net.ParseIP(dst)

							// parse host
							fqdn := parseHTTP.Host

							if fs.filterDomain(fqdn) || fs.filterConnPair(srcIP, dstIP) {
								continue
							}

							// disambiguate addresses which are not publicly routable
							srcUniqIP := data.NewUniqueIP(srcIP, parseHTTP.AgentUUID, parseHTTP.AgentHostname)
							dstUniqIP := data.NewUniqueIP(dstIP, parseHTTP.AgentUUID, parseHTTP.AgentHostname)
							srcProxyFQDNTrio := beaconproxy.NewUniqueSrcProxyHostnameTrio(srcUniqIP, dstUniqIP, fqdn)

							// get aggregation keys for ip addresses and connection pair
							srcProxyFQDNKey := srcProxyFQDNTrio.MapKey()

							// check if destination is a proxy server
							dstIsProxy := fs.checkIfProxyServer(dstIP)

							// parse method type
							method := parseHTTP.Method

							// check if internal IP is requesting a connection
							// through a proxy
							if method == "CONNECT" && dstIsProxy {

								// Safely store the number of conns for this uconn
								mutex.Lock()

								// add client (src) IP to hostname map

								// Check if the map value is set
								if _, ok := proxyHostnameMap[srcProxyFQDNKey]; !ok {
									// create new host record with src and dst
									proxyHostnameMap[srcProxyFQDNKey] = &beaconproxy.Input{
										Hosts: srcProxyFQDNTrio,
									}
								}

								// increment connection count
								proxyHostnameMap[srcProxyFQDNKey].ConnectionCount++

								// parse timestamp
								ts := parseHTTP.TimeStamp

								// add timestamp to unique timestamp list
								if !int64InSlice(ts, proxyHostnameMap[srcProxyFQDNKey].TsList) {
									proxyHostnameMap[srcProxyFQDNKey].TsList = append(proxyHostnameMap[srcProxyFQDNKey].TsList, ts)
								}

								mutex.Unlock()

							}

							// parse out useragent info
							userAgentName := parseHTTP.UserAgent
							if userAgentName == "" {
								userAgentName = "Empty user agent string"
							}

							// Safely store useragent information
							mutex.Lock()

							// create record if it doesn't exist
							if _, ok := useragentMap[userAgentName]; !ok {
								useragentMap[userAgentName] = &useragent.Input{
									Name:     userAgentName,
									Seen:     1,
									Requests: []string{fqdn},
								}
								useragentMap[userAgentName].OrigIps.Insert(srcUniqIP)
							} else {
								// increment times seen count
								useragentMap[userAgentName].Seen++

								// add src of useragent request to unique array
								useragentMap[userAgentName].OrigIps.Insert(srcUniqIP)

								// add request string to unique array
								if !stringInSlice(fqdn, useragentMap[userAgentName].Requests) {
									useragentMap[userAgentName].Requests = append(useragentMap[userAgentName].Requests, fqdn)
								}
							}

							mutex.Unlock()

						/// *************************************************************///
						///                           OPEN CONNS                         ///
						/// *************************************************************///
						case fs.config.T.Structure.OpenConnTable:
							parseConn, ok := datum.(*parsetypes.OpenConn)
							if !ok {
								continue
							}

							// get source destination pair for connection record
							src := parseConn.Source
							dst := parseConn.Destination

							// parse addresses into binary format
							srcIP := net.ParseIP(src)
							dstIP := net.ParseIP(dst)

							// disambiguate addresses which are not publicly routable
							srcUniqIP := data.NewUniqueIP(srcIP, parseConn.AgentUUID, parseConn.AgentHostname)
							dstUniqIP := data.NewUniqueIP(dstIP, parseConn.AgentUUID, parseConn.AgentHostname)
							srcDstPair := data.NewUniqueIPPair(srcUniqIP, dstUniqIP)

							// get aggregation keys for ip addresses and connection pair
							srcKey := srcUniqIP.MapKey()
							dstKey := dstUniqIP.MapKey()
							srcDstKey := srcDstPair.MapKey()

							// Run conn pair through filter to filter out certain connections
							ignore := fs.filterConnPair(srcIP, dstIP)

							// If connection pair is not subject to filtering, process
							if !ignore {
								ts := parseConn.TimeStamp
								origIPBytes := parseConn.OrigIPBytes
								respIPBytes := parseConn.RespIPBytes
								duration := parseConn.Duration
								duration = math.Ceil((duration)*10000) / 10000
								bytes := int64(origIPBytes + respIPBytes)
								protocol := parseConn.Proto
								service := parseConn.Service
								dstPort := parseConn.DestinationPort
								uid := parseConn.UID

								var tuple string
								if service == "" {
									tuple = strconv.Itoa(dstPort) + ":" + protocol + ":-"
								} else {
									tuple = strconv.Itoa(dstPort) + ":" + protocol + ":" + service
								}

								// Safely store the number of conns for this uconn
								mutex.Lock()

								// Check if the map value is set
								if _, ok := hostMap[srcKey]; !ok {
									// create new host record with src and dst
									hostMap[srcKey] = &host.Input{
										Host:    srcUniqIP,
										IsLocal: fs.checkIfInternal(srcIP),
										IP4:     util.IsIPv4(src),
										IP4Bin:  util.IPv4ToBinary(srcIP),
									}
								}

								// Check if the map value is set
								if _, ok := hostMap[dstKey]; !ok {
									// create new host record with src and dst
									hostMap[dstKey] = &host.Input{
										Host:    dstUniqIP,
										IsLocal: fs.checkIfInternal(dstIP),
										IP4:     util.IsIPv4(dst),
										IP4Bin:  util.IPv4ToBinary(dstIP),
									}
								}

								// Check if the map value is set
								if _, ok := uconnMap[srcDstKey]; !ok {
									// create new uconn record with src and dst
									// Set IsLocalSrc and IsLocalDst fields based on InternalSubnets setting
									// we only need to do this once if the uconn record does not exist
									uconnMap[srcDstKey] = &uconn.Input{
										Hosts:      srcDstPair,
										IsLocalSrc: fs.checkIfInternal(srcIP),
										IsLocalDst: fs.checkIfInternal(dstIP),
									}

									// Can do this even if the connection is open. For each set of logs we
									// process, we increment these values once per unique connection.
									// This might mean that an open connection has caused these values
									// to be incremented, but that is ok.
									hostMap[srcKey].CountSrc++
									hostMap[dstKey].CountDst++
								}

								// If the ConnStateList map doesn't exist for this entry, create it.
								if uconnMap[srcDstKey].ConnStateMap == nil {
									uconnMap[srcDstKey].ConnStateMap = make(map[string]*uconn.ConnState)
								}

								// If an entry for this open connection is present, first check
								// to make sure it hasn't been marked as closed. If it was marked
								// as closed, that supersedes any open connections information.
								// Otherwise, if it's open then check if this more up-to-date data
								// (i.e., longer duration)
								if _, ok := uconnMap[srcDstKey].ConnStateMap[uid]; ok {
									if (uconnMap[srcDstKey].ConnStateMap[uid].Open) && (duration > uconnMap[srcDstKey].ConnStateMap[uid].Duration) {
										uconnMap[srcDstKey].ConnStateMap[uid].Duration = duration

										// If current duration is longer than previous duration, we can
										// also assume that current bytes is /at least/ as big as the
										// stored value for bytes...same for OrigBytes
										uconnMap[srcDstKey].ConnStateMap[uid].Bytes = bytes
										uconnMap[srcDstKey].ConnStateMap[uid].OrigBytes = origIPBytes
									}
								} else {
									// No entry was present for a connection with this UID. Create a new
									// entry and set the Open state accordingly
									uconnMap[srcDstKey].ConnStateMap[uid] = &uconn.ConnState{
										Bytes:     bytes,
										Duration:  duration,
										Open:      true,
										OrigBytes: origIPBytes,
										Ts:        ts, //ts is the timestamp at which the connection was detected
										Tuple:     tuple,
									}
								}

								// this is to keep track of how many times a host connected to
								// an unexpected port - proto - service Tuple
								// we only want to increment the count once per unique destination,
								// not once per connection, hence the flag and the check
								if !uconnMap[srcDstKey].UPPSFlag {
									for _, entry := range trustedAppReferenceList {
										if (protocol == entry.protocol) && (dstPort == entry.port) {
											if service != entry.service {
												uconnMap[srcDstKey].UPPSFlag = true
											}
										}
									}
								}

								// increment unique dst port: proto : service tuple list for host.
								// Fine to do this with open connections as it's idempotent (yes, I like that word...)
								if !stringInSlice(tuple, uconnMap[srcDstKey].Tuples) {
									uconnMap[srcDstKey].Tuples = append(uconnMap[srcDstKey].Tuples, tuple)
								}

								// Check if invalid cert record was written before the uconns
								// record, we'll need to update it with the tuples.
								// Fine to do this with open connections as it's idempotent
								if _, ok := certMap[dstKey]; ok {
									// add tuple to invlaid cert list
									if !stringInSlice(tuple, certMap[dstKey].Tuples) {
										certMap[dstKey].Tuples = append(certMap[dstKey].Tuples, tuple)
									}
								}

								// Replace existing duration if current duration is higher. It's fine
								// to do this if the connection is still open as it will just get replaced
								// later when the connection closes. If an open connection is responsible
								// for the longest duration, we want to see that.
								if duration > uconnMap[srcDstKey].MaxDuration {
									uconnMap[srcDstKey].MaxDuration = duration
								}

								if duration > hostMap[srcKey].MaxDuration {
									hostMap[srcKey].MaxDuration = duration
								}
								if duration > hostMap[dstKey].MaxDuration {
									hostMap[dstKey].MaxDuration = duration
								}

								// NOTE: We are not incrementing the connection counters until the
								// connection closes to prevent double-counting.

								mutex.Unlock()

							}

						/// *************************************************************///
						///                             SSL                              ///
						/// *************************************************************///
						case fs.config.T.Structure.SSLTable:
							parseSSL, ok := datum.(*parsetypes.SSL)
							if !ok {
								continue
							}
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
							mutex.Lock()

							// create useragent record if it doesn't exist
							if _, ok := useragentMap[ja3Hash]; !ok {
								useragentMap[ja3Hash] = &useragent.Input{
									Name:     ja3Hash,
									Seen:     1,
									Requests: []string{host},
									JA3:      true,
								}
								useragentMap[ja3Hash].OrigIps.Insert(srcUniqIP)
							} else {
								// increment times seen count
								useragentMap[ja3Hash].Seen++

								// add src of ssl request to unique array
								useragentMap[ja3Hash].OrigIps.Insert(srcUniqIP)

								// add request string to unique array
								if !stringInSlice(host, useragentMap[ja3Hash].Requests) {
									useragentMap[ja3Hash].Requests = append(useragentMap[ja3Hash].Requests, host)
								}
							}

							// create uconn and cert records
							// Run conn pair through filter to filter out certain connections
							ignore := fs.filterConnPair(srcIP, dstIP)
							if !ignore {

								// Check if uconn map value is set, because this record could
								// come before a relevant uconns record (or may be the only source
								// for the uconns record)
								if _, ok := uconnMap[srcDstKey]; !ok {
									// create new uconn record if it does not exist
									uconnMap[srcDstKey] = &uconn.Input{
										Hosts:      srcDstPair,
										IsLocalSrc: fs.checkIfInternal(srcIP),
										IsLocalDst: fs.checkIfInternal(dstIP),
									}
								}

								//if there's any problem in the certificate, mark it invalid
								if certStatus != "ok" && certStatus != "-" && certStatus != "" && certStatus != " " {
									// mark as having invalid cert
									uconnMap[srcDstKey].InvalidCertFlag = true

									// update relevant cert record
									if _, ok := certMap[dstKey]; !ok {
										// create new uconn record if it does not exist
										certMap[dstKey] = &certificate.Input{
											Host: dstUniqIP,
											Seen: 1,
										}
									} else {
										certMap[dstKey].Seen++
									}

									for _, tuple := range uconnMap[srcDstKey].Tuples {
										// mark as having invalid cert
										if !stringInSlice(tuple, certMap[dstKey].Tuples) {
											certMap[dstKey].Tuples = append(certMap[dstKey].Tuples, tuple)
										}
									}
									// mark as having invalid cert
									if !stringInSlice(certStatus, certMap[dstKey].InvalidCerts) {
										certMap[dstKey].InvalidCerts = append(certMap[dstKey].InvalidCerts, certStatus)
									}
									// add src of ssl request to unique array
									certMap[dstKey].OrigIps.Insert(srcUniqIP)
								}
							}

							mutex.Unlock()
						}
					}
				}
				indexedFiles[j].ParseTime = time.Now()
				fileHandle.Close()
				logger.WithFields(log.Fields{
					"path": indexedFiles[j].Path,
				}).Info("Finished parsing file")
			}
			wg.Done()
		}(indexedFiles, logger, parsingWG, i, parsingThreads, n)
	}
	parsingWG.Wait()

	return uconnMap, hostMap, explodeddnsMap, hostnameMap, proxyHostnameMap, useragentMap, certMap
}

//buildExplodedDNS .....
func (fs *FSImporter) buildExplodedDNS(domainMap map[string]int) {

	if fs.config.S.DNS.Enabled {
		if len(domainMap) > 0 {
			// Set up the database
			explodedDNSRepo := explodeddns.NewMongoRepository(fs.database, fs.config, fs.log)
			err := explodedDNSRepo.CreateIndexes()
			if err != nil {
				fs.log.Error(err)
			}
			explodedDNSRepo.Upsert(domainMap)
		} else {
			fmt.Println("\t[!] No DNS data to analyze")
		}
	}
}

//buildCertificates .....
func (fs *FSImporter) buildCertificates(certMap map[string]*certificate.Input) {

	if len(certMap) > 0 {
		// Set up the database
		certificateRepo := certificate.NewMongoRepository(fs.database, fs.config, fs.log)
		err := certificateRepo.CreateIndexes()
		if err != nil {
			fs.log.Error(err)
		}
		certificateRepo.Upsert(certMap)
	} else {
		fmt.Println("\t[!] No invalid certificate data to analyze")
	}

}

//removeAnalysisChunk .....
func (fs *FSImporter) removeAnalysisChunk(cid int) error {

	// Set up the remover
	removerRepo := remover.NewMongoRemover(fs.database, fs.config, fs.log)
	err := removerRepo.Remove(cid)
	if err != nil {
		fs.log.Error(err)
		return err
	}

	fs.metaDB.SetChunk(cid, fs.database.GetSelectedDB(), false)

	return nil

}

//buildHostnames .....
func (fs *FSImporter) buildHostnames(hostnameMap map[string]*hostname.Input) {
	// non-optional module
	if len(hostnameMap) > 0 {
		// Set up the database
		hostnameRepo := hostname.NewMongoRepository(fs.database, fs.config, fs.log)
		err := hostnameRepo.CreateIndexes()
		if err != nil {
			fs.log.Error(err)
		}
		hostnameRepo.Upsert(hostnameMap)
	} else {
		fmt.Println("\t[!] No Hostname data to analyze")
	}

}

func (fs *FSImporter) buildUconns(uconnMap map[string]*uconn.Input) {
	// non-optional module
	if len(uconnMap) > 0 {
		// Set up the database
		uconnRepo := uconn.NewMongoRepository(fs.database, fs.config, fs.log)

		err := uconnRepo.CreateIndexes()
		if err != nil {
			fs.log.Error(err)
		}

		// send uconns to uconn analysis
		uconnRepo.Upsert(uconnMap)
	} else {
		fmt.Println("\t[!] No Uconn data to analyze")
		fmt.Printf("\t\t[!!] No local network traffic found, please check ")
		fmt.Println("InternalSubnets in your RITA config (/etc/rita/config.yaml)")
	}
}

func (fs *FSImporter) buildHosts(hostMap map[string]*host.Input) {
	// non-optional module
	if len(hostMap) > 0 {
		hostRepo := host.NewMongoRepository(fs.database, fs.config, fs.log)

		err := hostRepo.CreateIndexes()
		if err != nil {
			fs.log.Error(err)
		}

		// send uconns to host analysis
		hostRepo.Upsert(hostMap)
	} else {
		fmt.Println("\t[!] No Host data to analyze")
		fmt.Printf("\t\t[!!] No local network traffic found, please check ")
		fmt.Println("InternalSubnets in your RITA config (/etc/rita/config.yaml)")
	}
}

func (fs *FSImporter) markBlacklistedPeers(hostMap map[string]*host.Input) {
	// non-optional module
	if len(hostMap) > 0 {
		blacklistRepo := blacklist.NewMongoRepository(fs.database, fs.config, fs.log)

		err := blacklistRepo.CreateIndexes()
		if err != nil {
			fs.log.Error(err)
		}

		// send uconns to host analysis
		blacklistRepo.Upsert()
	}
}

func (fs *FSImporter) buildBeacons(uconnMap map[string]*uconn.Input, minTimestamp, maxTimestamp int64) {
	if fs.config.S.Beacon.Enabled {
		if len(uconnMap) > 0 {
			beaconRepo := beacon.NewMongoRepository(fs.database, fs.config, fs.log)

			err := beaconRepo.CreateIndexes()
			if err != nil {
				fs.log.Error(err)
			}

			// send uconns to beacon analysis
			beaconRepo.Upsert(uconnMap, minTimestamp, maxTimestamp)
		} else {
			fmt.Println("\t[!] No Beacon data to analyze")
		}
	}

}

func (fs *FSImporter) buildFQDNBeacons(hostnameMap map[string]*hostname.Input, minTimestamp, maxTimestamp int64) {
	if fs.config.S.BeaconFQDN.Enabled {
		if len(hostnameMap) > 0 {
			beaconFQDNRepo := beaconfqdn.NewMongoRepository(fs.database, fs.config, fs.log)

			err := beaconFQDNRepo.CreateIndexes()
			if err != nil {
				fs.log.Error(err)
			}

			// send uconns to beacon analysis
			beaconFQDNRepo.Upsert(hostnameMap, minTimestamp, maxTimestamp)
		} else {
			fmt.Println("\t[!] No FQDN Beacon data to analyze")
		}
	}

}

func (fs *FSImporter) buildProxyBeacons(proxyHostnameMap map[string]*beaconproxy.Input, minTimestamp, maxTimestamp int64) {
	if fs.config.S.BeaconProxy.Enabled {
		if len(proxyHostnameMap) > 0 {
			beaconProxyRepo := beaconproxy.NewMongoRepository(fs.database, fs.config, fs.log)

			err := beaconProxyRepo.CreateIndexes()
			if err != nil {
				fs.log.Error(err)
			}

			// send uconns to beacon analysis
			beaconProxyRepo.Upsert(proxyHostnameMap, minTimestamp, maxTimestamp)
		} else {
			fmt.Println("\t[!] No Proxy Beacon data to analyze")
		}
	}

}

//buildUserAgent .....
func (fs *FSImporter) buildUserAgent(useragentMap map[string]*useragent.Input) {

	if fs.config.S.UserAgent.Enabled {
		if len(useragentMap) > 0 {
			// Set up the database
			useragentRepo := useragent.NewMongoRepository(fs.database, fs.config, fs.log)

			err := useragentRepo.CreateIndexes()
			if err != nil {
				fs.log.Error(err)
			}
			useragentRepo.Upsert(useragentMap)
		} else {
			fmt.Println("\t[!] No UserAgent data to analyze")
		}
	}
}

func (fs *FSImporter) updateTimestampRange() (int64, int64) {
	session := fs.database.Session.Copy()
	defer session.Close()

	// set collection name
	collectionName := fs.config.T.Structure.UniqueConnTable

	// check if collection already exists
	names, _ := session.DB(fs.database.GetSelectedDB()).CollectionNames()

	exists := false
	// make sure collection exists
	for _, name := range names {
		if name == collectionName {
			exists = true
			break
		}
	}

	if !exists {
		return 0, 0
	}

	// Build query for aggregation
	timestampMinQuery := []bson.M{
		{"$project": bson.M{
			"_id":     0,
			"ts":      "$dat.ts",
			"open_ts": bson.M{"$ifNull": []interface{}{"$open_ts", []interface{}{}}},
		}},
		{"$unwind": "$ts"},
		{"$project": bson.M{"_id": 0, "ts": bson.M{"$concatArrays": []interface{}{"$ts", "$open_ts"}}}},
		{"$unwind": "$ts"}, // Not an error, must unwind it twice
		{"$sort": bson.M{"ts": 1}},
		{"$limit": 1},
	}

	var resultMin struct {
		Timestamp int64 `bson:"ts"`
	}

	// get iminimum timestamp
	// sort by the timestamp, limit it to 1 (only returns first result)
	err := session.DB(fs.database.GetSelectedDB()).C(collectionName).Pipe(timestampMinQuery).AllowDiskUse().One(&resultMin)

	if err != nil {
		fs.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not retrieve minimum timestamp:", err)
		return 0, 0
	}

	// Build query for aggregation
	timestampMaxQuery := []bson.M{
		{"$project": bson.M{
			"_id":     0,
			"ts":      "$dat.ts",
			"open_ts": bson.M{"$ifNull": []interface{}{"$open_ts", []interface{}{}}},
		}},
		{"$unwind": "$ts"},
		{"$project": bson.M{"_id": 0, "ts": bson.M{"$concatArrays": []interface{}{"$ts", "$open_ts"}}}},
		{"$unwind": "$ts"}, // Not an error, must unwind it twice
		{"$sort": bson.M{"ts": -1}},
		{"$limit": 1},
	}

	var resultMax struct {
		Timestamp int64 `bson:"ts"`
	}

	// get max timestamp
	// sort by the timestamp, limit it to 1 (only returns first result)
	err = session.DB(fs.database.GetSelectedDB()).C(collectionName).Pipe(timestampMaxQuery).AllowDiskUse().One(&resultMax)

	if err != nil {
		fs.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not retrieve maximum timestamp:", err)
		return 0, 0
	}

	// set range in metadatabase
	err = fs.metaDB.AddTSRange(fs.database.GetSelectedDB(), resultMin.Timestamp, resultMax.Timestamp)
	if err != nil {
		fs.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not set ts range in metadatabase: ", err)
		return 0, 0
	}
	return resultMin.Timestamp, resultMax.Timestamp
}

//stringInSlice ...
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

//int64InSlice ...
func int64InSlice(a int64, list []int64) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
