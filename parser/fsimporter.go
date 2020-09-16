package parser

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	fpt "github.com/activecm/rita/parser/fileparsetypes"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/beacon"
	"github.com/activecm/rita/pkg/blacklist"
	"github.com/activecm/rita/pkg/certificate"
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
		res             *resources.Resources
		importFiles     []string
		rolling         bool
		totalChunks     int
		currentChunk    int
		indexingThreads int
		parseThreads    int
		batchSizeBytes  int64
		internal        []*net.IPNet
		alwaysIncluded  []*net.IPNet
		neverIncluded   []*net.IPNet
	}

	trustedAppTiplet struct {
		protocol string
		port     int
		service  string
	}
)

//NewFSImporter creates a new file system importer
func NewFSImporter(res *resources.Resources,
	indexingThreads int, parseThreads int, importFiles []string) *FSImporter {
	return &FSImporter{
		res:             res,
		importFiles:     importFiles,
		rolling:         res.Config.S.Rolling.Rolling,
		totalChunks:     res.Config.S.Rolling.TotalChunks,
		currentChunk:    res.Config.S.Rolling.CurrentChunk,
		indexingThreads: indexingThreads,
		parseThreads:    parseThreads,
		batchSizeBytes:  2 * (2 << 30), // 2 gigabytes (used to not run out of memory while importing)
		internal:        getParsedSubnets(res.Config.S.Filtering.InternalSubnets),
		alwaysIncluded:  getParsedSubnets(res.Config.S.Filtering.AlwaysInclude),
		neverIncluded:   getParsedSubnets(res.Config.S.Filtering.NeverInclude),
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

func (fs *FSImporter) CollectFileDetails() []*fpt.IndexedFile {
	// find all of the potential bro log paths
	files := readFiles(fs.importFiles, fs.res.Log)

	// hash the files and get their stats
	return indexFiles(files, fs.indexingThreads, fs.res)
}

//Run starts the importing
func (fs *FSImporter) Run(indexedFiles []*fpt.IndexedFile) {
	start := time.Now()

	fmt.Println("\t[-] Verifying log files have not been previously parsed into the target dataset ... ")
	// check list of files against metadatabase records to ensure that the a file
	// won't be imported into the same database twice.
	indexedFiles = removeOldFilesFromIndex(indexedFiles, fs.res.MetaDB, fs.res.Log, fs.res.DB.GetSelectedDB())

	// if all files were removed because they've already been imported, handle error
	if !(len(indexedFiles) > 0) {
		if fs.rolling {
			fmt.Println("\t[!] All files pertaining to the current chunk entry have already been parsed into database: ", fs.res.DB.GetSelectedDB())
		} else {
			fmt.Println("\t[!] All files in this directory have already been parsed into database: ", fs.res.DB.GetSelectedDB())
		}
		return
	}

	// Add new metadatabase record for db if doesn't already exist
	dbExists, err := fs.res.MetaDB.DBExists(fs.res.DB.GetSelectedDB())
	if err != nil {
		fs.res.Log.WithFields(log.Fields{
			"err":      err,
			"database": fs.res.DB.GetSelectedDB(),
		}).Error("Could not check if metadatabase record exists for target database")
		fmt.Printf("\t[!] %v", err.Error())
	}

	if !dbExists {
		err := fs.res.MetaDB.AddNewDB(fs.res.DB.GetSelectedDB(), fs.currentChunk, fs.totalChunks)
		if err != nil {
			fs.res.Log.WithFields(log.Fields{
				"err":      err,
				"database": fs.res.DB.GetSelectedDB(),
			}).Error("Could not add metadatabase record for new database")
			fmt.Printf("\t[!] %v", err.Error())
		}
	}

	if fs.rolling {
		err := fs.res.MetaDB.SetRollingSettings(fs.res.DB.GetSelectedDB(), fs.currentChunk, fs.totalChunks)
		if err != nil {
			fs.res.Log.WithFields(log.Fields{
				"err":      err,
				"database": fs.res.DB.GetSelectedDB(),
			}).Error("Could not update rolling database settings for database")
			fmt.Printf("\t[!] %v", err.Error())
		}

		chunkSet, err := fs.res.MetaDB.IsChunkSet(fs.currentChunk, fs.res.DB.GetSelectedDB())
		if err != nil {
			fmt.Println("\t[!] Could not find CID List entry in metadatabase")
			return
		}

		if chunkSet {
			fmt.Println("\t[-] Removing outdated data from rolling dataset ... ")
			err := fs.removeAnalysisChunk(fs.currentChunk)
			if err != nil {
				fmt.Println("\t[!] Failed to remove outdata data from rolling dataset")
				return
			}
		}
	}

	// create blacklisted reference Collection if blacklisted module is enabled
	if fs.res.Config.S.Blacklisted.Enabled {
		blacklist.BuildBlacklistedCollections(fs.res)
	}

	// batch up the indexed files so as not to read too much in at one time
	batchedIndexedFiles := batchFilesBySize(indexedFiles, fs.batchSizeBytes)

	for i, indexedFileBatch := range batchedIndexedFiles {
		fmt.Printf("\t[-] Processing batch %d of %d\n", i+1, len(batchedIndexedFiles))

		// parse in those files!
		uconnMap, hostMap, explodeddnsMap, hostnameMap, useragentMap, certMap := fs.parseFiles(indexedFileBatch, fs.parseThreads, fs.res.Log)

		// Set chunk before we continue so if process dies, we still verify with a delete if
		// any data was written out.
		fs.res.MetaDB.SetChunk(fs.currentChunk, fs.res.DB.GetSelectedDB(), true)

		// build Hosts table.
		fs.buildHosts(hostMap)

		// build Uconns table. Must go before beacons.
		fs.buildUconns(uconnMap)

		// update ts range for dataset (needs to be run before beacons)
		fs.updateTimestampRange()

		// build or update the exploded DNS table. Must go before hostnames
		fs.buildExplodedDNS(explodeddnsMap)

		// build or update the exploded DNS table
		fs.buildHostnames(hostnameMap)

		// build or update Beacons table
		fs.buildBeacons(uconnMap)

		// build or update UserAgent table
		fs.buildUserAgent(useragentMap)

		// build or update Certificate table
		fs.buildCertificates(certMap)

		// update blacklisted peers in hosts collection
		fs.markBlacklistedPeers(hostMap)

		// record file+database name hash in metadabase to prevent duplicate content
		fmt.Println("\t[-] Indexing log entries ... ")
		updateFilesIndex(indexedFileBatch, fs.res.MetaDB, fs.res.Log)
	}

	// mark results as imported and analyzed
	fmt.Println("\t[-] Updating metadatabase ... ")
	fs.res.MetaDB.MarkDBAnalyzed(fs.res.DB.GetSelectedDB(), true)

	progTime := time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"current_time": progTime.Format(util.TimeFormat),
			"total_time":   progTime.Sub(start).String(),
		},
	).Info("Finished upload. Starting indexing")

	progTime = time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"current_time": progTime.Format(util.TimeFormat),
			"total_time":   progTime.Sub(start).String(),
		},
	).Info("Finished importing log files")

	fmt.Println("\t[-] Done!")
}

// batchFilesBySize takes in an slice of indexedFiles and splits the array into
// subgroups of indexedFiles such that each group has a total size in bytes less than size
func batchFilesBySize(indexedFiles []*fpt.IndexedFile, size int64) [][]*fpt.IndexedFile {
	// sort the indexed files so we process them in order
	sort.Slice(indexedFiles, func(i, j int) bool {
		return indexedFiles[i].Path < indexedFiles[j].Path
	})

	//group by target collection
	fileTypeMap := make(map[string][]*fpt.IndexedFile)
	for _, file := range indexedFiles {
		if _, ok := fileTypeMap[file.TargetCollection]; !ok {
			fileTypeMap[file.TargetCollection] = make([]*fpt.IndexedFile, 0)
		}
		fileTypeMap[file.TargetCollection] = append(fileTypeMap[file.TargetCollection], file)
	}

	// Take n files in each target collection group until the size limit is exceeded, then start a new batch
	batches := make([][]*fpt.IndexedFile, 0)
	currBatch := make([]*fpt.IndexedFile, 0)
	currAggBytes := int64(0)
	iterators := make(map[string]int)
	for fileType := range fileTypeMap {
		iterators[fileType] = 0
	}

	for len(iterators) != 0 { // while there is data to iterate through

		maybeBatch := make([]*fpt.IndexedFile, 0)
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
			currBatch = make([]*fpt.IndexedFile, 0)
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
func (fs *FSImporter) parseFiles(indexedFiles []*fpt.IndexedFile, parsingThreads int, logger *log.Logger) (
	map[string]*uconn.Pair, map[string]*host.IP, map[string]int, map[string]*hostname.Input, map[string]*useragent.Input, map[string]*certificate.Input) {

	fmt.Println("\t[-] Parsing logs to: " + fs.res.DB.GetSelectedDB() + " ... ")

	//TODO[AGENT]: Create struct type for map keys which contains network ids
	/*
		//perhaps place this in pkg/data and embed it in host.IP, certificate.Input, and uconn.Pair
		// might need New(...) constructor to properly zero out NetworkID if we don't have network data
		type UniqueIP struct {
			IP string
			NetworkID bson.Binary //used for efficient UUID
			NetworkName string
		}
		// only use IP and NetworkID.Data to compare UniqueIPs
		func (u UniqueIP) hash() uint64 {
			hasher := fnv.New64a()
			hasher.Write(*(*[]byte)(unsafe.Pointer(&u.IP)))
			hasher.Write(u.NetworkID.Data) //Needs to work in case this is nil/zero [standard zeek install]
			return hasher.Sum64()
		}
		func (u UniqueIP) hashWith(other UniqueIP) uint64 {
			hasher := fnv.New64a()
			hasher.Write(*(*[]byte)(unsafe.Pointer(&u.IP)))
			hasher.Write(u.NetworkID.Data)
			hasher.Write(*(*[]byte)(unsafe.Pointer(&other.IP)))
			hasher.Write(other.NetworkID.Data)
			return hasher.Sum64()
		}

	*/

	// create log parsing maps
	explodeddnsMap := make(map[string]int)

	hostnameMap := make(map[string]*hostname.Input)

	useragentMap := make(map[string]*useragent.Input)

	certMap := make(map[string]*certificate.Input)

	// Counts the number of uconns per source-destination pair
	uconnMap := make(map[string]*uconn.Pair)

	// Counts the number of uconns per source-destination pair
	hostMap := make(map[string]*host.IP)

	//set up parallel parsing
	n := len(indexedFiles)
	parsingWG := new(sync.WaitGroup)

	// Creates a mutex for locking map keys during read-write operations
	var mutex = &sync.Mutex{}

	for i := 0; i < parsingThreads; i++ {
		parsingWG.Add(1)

		go func(indexedFiles []*fpt.IndexedFile, logger *log.Logger,
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
				fileScanner, err := getFileScanner(fileHandle)
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
					data := parseLine(
						fileScanner.Text(),
						indexedFiles[j].GetHeader(),
						indexedFiles[j].GetFieldMap(),
						indexedFiles[j].GetBroDataFactory(),
						indexedFiles[j].IsJSON(),
						logger,
					)

					if data != nil {
						//figure out which collection (dns, http, or conn) this line is heading for
						targetCollection := indexedFiles[j].TargetCollection

						switch targetCollection {

						/// *************************************************************///
						///                           CONNS                              ///
						/// *************************************************************///
						case fs.res.Config.T.Structure.ConnTable:

							parseConn, ok := data.(*parsetypes.Conn)
							if !ok {
								continue
							}

							// get source destination pair for connection record
							src := parseConn.Source
							dst := parseConn.Destination

							// Run conn pair through filter to filter out certain connections
							ignore := fs.filterConnPair(src, dst)

							// If connection pair is not subject to filtering, process
							if !ignore {
								//TODO[AGENT]: grab network ids / names for source and destination
								//TODO[AGENT]: build UniqueIP objects from IPs and network ids and names
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

								//TODO[AGENT]: Instead of keying uconn map in on src + dst string, key on UniqueIP.hashWith()
								// Concatenate the source and destination IPs to use as a map key
								srcDst := src + dst

								// Safely store the number of conns for this uconn
								mutex.Lock()

								//TODO[AGENT]: Instead of keying hostmap in on ip string, key on UniqueIP.hash()
								// Check if the map value is set
								if _, ok := hostMap[src]; !ok {
									// create new host record with src and dst
									hostMap[src] = &host.IP{
										Host:    src,
										IsLocal: containsIP(fs.GetInternalSubnets(), net.ParseIP(src)),
										IP4:     isIPv4(src),
										IP4Bin:  ipv4ToBinary(net.ParseIP(src)),
									}
								}

								// Check if the map value is set
								if _, ok := hostMap[dst]; !ok {
									// create new host record with src and dst
									hostMap[dst] = &host.IP{
										Host:    dst,
										IsLocal: containsIP(fs.GetInternalSubnets(), net.ParseIP(dst)),
										IP4:     isIPv4(dst),
										IP4Bin:  ipv4ToBinary(net.ParseIP(dst)),
									}
								}

								// Check if the map value is set
								if _, ok := uconnMap[srcDst]; !ok {
									// create new uconn record with src and dst
									// Set IsLocalSrc and IsLocalDst fields based on InternalSubnets setting
									// we only need to do this once if the uconn record does not exist
									uconnMap[srcDst] = &uconn.Pair{
										Src:        src,
										Dst:        dst,
										IsLocalSrc: containsIP(fs.GetInternalSubnets(), net.ParseIP(src)),
										IsLocalDst: containsIP(fs.GetInternalSubnets(), net.ParseIP(dst)),
									}

									hostMap[src].CountSrc++
									hostMap[dst].CountDst++
								}

								// this is to keep track of how many times a host connected to
								// an unexpected port - proto - service Tuple
								// we only want to increment the count once per unique destination,
								// not once per connection, hence the flag and the check
								if uconnMap[srcDst].UPPSFlag == false {
									for _, entry := range trustedAppReferenceList {
										if (protocol == entry.protocol) && (dstPort == entry.port) {
											if service != entry.service {
												hostMap[src].UntrustedAppConnCount++
												uconnMap[srcDst].UPPSFlag = true
											}
										}
									}
								}

								// increment unique dst port: proto : service tuple list for host
								if stringInSlice(tuple, uconnMap[srcDst].Tuples) == false {
									uconnMap[srcDst].Tuples = append(uconnMap[srcDst].Tuples, tuple)
								}

								//TODO[AGENT]: Instead of keying certMap in on ip string, key on UniqueIP.hash()

								// Check if invalid cert record was written before the uconns
								// record, we'll need to update it with the tuples.
								if _, ok := certMap[dst]; ok {
									// add tuple to invlaid cert list
									if stringInSlice(tuple, certMap[dst].Tuples) == false {
										certMap[dst].Tuples = append(certMap[dst].Tuples, tuple)
									}
								}

								// Increment the connection count for the src-dst pair
								uconnMap[srcDst].ConnectionCount++
								hostMap[src].ConnectionCount++
								hostMap[dst].ConnectionCount++

								//TODO[AGENT]: Covnert IP.ConnectedDstHosts to map[string]UniqueIP
								if stringInSlice(dst, hostMap[src].ConnectedDstHosts) == false {
									hostMap[src].ConnectedDstHosts = append(hostMap[src].ConnectedDstHosts, dst)
								}

								//TODO[AGENT]: Covnert IP.ConnectedSrcHosts to map[string]UniqueIP
								if stringInSlice(src, hostMap[dst].ConnectedSrcHosts) == false {
									hostMap[dst].ConnectedSrcHosts = append(hostMap[dst].ConnectedSrcHosts, src)
								}

								// Only append unique timestamps to tslist
								if int64InSlice(ts, uconnMap[srcDst].TsList) == false {
									uconnMap[srcDst].TsList = append(uconnMap[srcDst].TsList, ts)
								}

								// Append all origIPBytes to origBytesList
								uconnMap[srcDst].OrigBytesList = append(uconnMap[srcDst].OrigBytesList, origIPBytes)

								// Calculate and store the total number of bytes exchanged by the uconn pair
								uconnMap[srcDst].TotalBytes += bytes
								hostMap[src].TotalBytes += bytes
								hostMap[dst].TotalBytes += bytes

								// Calculate and store the total duration
								uconnMap[srcDst].TotalDuration += duration
								hostMap[src].TotalDuration += duration
								hostMap[dst].TotalDuration += duration

								// Replace existing duration if current duration is higher
								if duration > uconnMap[srcDst].MaxDuration {
									uconnMap[srcDst].MaxDuration = duration
								}

								if duration > hostMap[src].MaxDuration {
									hostMap[src].MaxDuration = duration
								}
								if duration > hostMap[dst].MaxDuration {
									hostMap[dst].MaxDuration = duration
								}

								mutex.Unlock()

							}

							/// *************************************************************///
							///                             DNS                             ///
							/// *************************************************************///
						case fs.res.Config.T.Structure.DNSTable:
							parseDNS, ok := data.(*parsetypes.DNS)
							if !ok {
								continue
							}

							domain := parseDNS.Query
							queryTypeName := parseDNS.QTypeName

							// Safely store the number of conns for this uconn
							mutex.Lock()

							// increment domain map count for exploded dns
							explodeddnsMap[domain]++

							// initialize the hostname input objects for new hostnames
							if _, ok := hostnameMap[domain]; !ok {
								hostnameMap[domain] = &hostname.Input{}
							}

							// geo.vortex.data.microsoft.com.akadns.net

							//TODO[AGENT]: Use UniqueIP/ NetworkID in hostnameMap ClientIPs

							// extract and store the dns client ip address
							src := parseDNS.Source
							if stringInSlice(src, hostnameMap[domain].ClientIPs) == false {
								hostnameMap[domain].ClientIPs = append(hostnameMap[domain].ClientIPs, src)
							}

							//TODO[AGENT]: Use UniqueIP/ NetworkID in hostnameMap ResolvedIPs

							if queryTypeName == "A" {
								answers := parseDNS.Answers
								for _, answer := range answers {
									// Check if answer is an IP address and store it if it is
									if net.ParseIP(answer) != nil {
										if stringInSlice(answer, hostnameMap[domain].ResolvedIPs) == false {
											hostnameMap[domain].ResolvedIPs = append(hostnameMap[domain].ResolvedIPs, answer)
										}
									}
								}
							}

							// increment txt query count for host in uconn
							if queryTypeName == "TXT" {
								// get destination for dns record
								dst := parseDNS.Destination

								// Run conn pair through filter to filter out certain connections
								ignore := fs.filterConnPair(src, dst)
								if !ignore {

									//TODO[AGENT]: Index hostmap with UniqueIP hash rather than src IP string in DNS parsing

									// Check if host map value is set, because this record could
									// come before a relevant conns record
									if _, ok := hostMap[src]; !ok {
										// create new uconn record with src and dst
										// Set IsLocalSrc and IsLocalDst fields based on InternalSubnets setting
										// we only need to do this once if the uconn record does not exist
										hostMap[src] = &host.IP{
											Host:    src,
											IsLocal: containsIP(fs.GetInternalSubnets(), net.ParseIP(src)),
											IP4:     isIPv4(src),
											IP4Bin:  ipv4ToBinary(net.ParseIP(src)),
										}
									}
									// increment txt query count
									hostMap[src].TXTQueryCount++
								}

							}

							mutex.Unlock()

							/// *************************************************************///
							///                             HTTP                             ///
							/// *************************************************************///
						case fs.res.Config.T.Structure.HTTPTable:
							parseHTTP, ok := data.(*parsetypes.HTTP)
							if !ok {
								continue
							}
							userAgentName := parseHTTP.UserAgent
							src := parseHTTP.Source
							host := parseHTTP.Host

							if userAgentName == "" {
								userAgentName = "Empty user agent string"
							}

							// Safely store useragent information
							mutex.Lock()

							//TODO[AGENT]: Use UniqueIP with NetworkID for OrigIPs in useragentMap
							// create record if it doesn't exist
							if _, ok := useragentMap[userAgentName]; !ok {
								useragentMap[userAgentName] = &useragent.Input{Seen: 1, OrigIps: []string{src}, Requests: []string{host}}
							} else {
								// increment times seen count
								useragentMap[userAgentName].Seen++

								// add src of useragent request to unique array
								if stringInSlice(src, useragentMap[userAgentName].OrigIps) == false {
									useragentMap[userAgentName].OrigIps = append(useragentMap[userAgentName].OrigIps, src)
								}

								// add request string to unique array
								if stringInSlice(host, useragentMap[userAgentName].Requests) == false {
									useragentMap[userAgentName].Requests = append(useragentMap[userAgentName].Requests, host)
								}
							}

							mutex.Unlock()

							/// *************************************************************///
							///                             SSL                             ///
							/// *************************************************************///
						case fs.res.Config.T.Structure.SSLTable:
							parseSSL, ok := data.(*parsetypes.SSL)
							if !ok {
								continue
							}
							ja3Hash := parseSSL.JA3
							src := parseSSL.Source
							dst := parseSSL.Destination
							host := parseSSL.ServerName
							certStatus := parseSSL.ValidationStatus

							if ja3Hash == "" {
								ja3Hash = "No JA3 hash generated"
							}

							// Safely store ja3 information
							mutex.Lock()

							//TODO[AGENT]: Use UniqueIP with NetworkID for OrigIPs in useragentMap
							// create record if it doesn't exist
							if _, ok := useragentMap[ja3Hash]; !ok {
								useragentMap[ja3Hash] = &useragent.Input{
									Seen:     1,
									OrigIps:  []string{src},
									Requests: []string{host},
									JA3:      true,
								}
							} else {
								// increment times seen count
								useragentMap[ja3Hash].Seen++

								// add src of ssl request to unique array
								if stringInSlice(src, useragentMap[ja3Hash].OrigIps) == false {
									useragentMap[ja3Hash].OrigIps = append(useragentMap[ja3Hash].OrigIps, src)
								}

								// add request string to unique array
								if stringInSlice(host, useragentMap[ja3Hash].Requests) == false {
									useragentMap[ja3Hash].Requests = append(useragentMap[ja3Hash].Requests, host)
								}
							}

							//if there's any problem in the certificate, mark it invalid
							if certStatus != "ok" && certStatus != "-" && certStatus != "" && certStatus != " " {
								// Run conn pair through filter to filter out certain connections
								ignore := fs.filterConnPair(src, dst)
								if !ignore {

									//TODO[AGENT]: Index uconnMap with UniqueIP hashWith rather than src+dst string in SSL parsing

									// Check if uconn map value is set, because this record could
									// come before a relevant uconns record
									if _, ok := uconnMap[src+dst]; !ok {
										// create new uconn record if it does not exist
										uconnMap[src+dst] = &uconn.Pair{
											Src:        src,
											Dst:        dst,
											IsLocalSrc: containsIP(fs.GetInternalSubnets(), net.ParseIP(src)),
											IsLocalDst: containsIP(fs.GetInternalSubnets(), net.ParseIP(dst)),
										}
									}
									// mark as having invalid cert
									uconnMap[src+dst].InvalidCertFlag = true

									//TODO[AGENT]: Index certMap with UniqueIP hashh rather than dst IP string in SSL parsing

									// update relevant cert record
									if _, ok := certMap[dst]; !ok {
										// create new uconn record if it does not exist
										certMap[dst] = &certificate.Input{
											Host: dst,
											Seen: 1,
										}
									} else {
										certMap[dst].Seen++
									}

									for _, tuple := range uconnMap[src+dst].Tuples {
										// mark as having invalid cert
										if stringInSlice(tuple, certMap[dst].Tuples) == false {
											certMap[dst].Tuples = append(certMap[dst].Tuples, tuple)
										}
									}
									// mark as having invalid cert
									if stringInSlice(certStatus, certMap[dst].InvalidCerts) == false {
										certMap[dst].InvalidCerts = append(certMap[dst].InvalidCerts, certStatus)
									}
									// add src of ssl request to unique array
									if stringInSlice(src, certMap[dst].OrigIps) == false {
										certMap[dst].OrigIps = append(certMap[dst].OrigIps, src)
									}
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

	return uconnMap, hostMap, explodeddnsMap, hostnameMap, useragentMap, certMap
}

//buildExplodedDNS .....
func (fs *FSImporter) buildExplodedDNS(domainMap map[string]int) {

	if fs.res.Config.S.DNS.Enabled {
		if len(domainMap) > 0 {
			// Set up the database
			explodedDNSRepo := explodeddns.NewMongoRepository(fs.res)
			err := explodedDNSRepo.CreateIndexes()
			if err != nil {
				fs.res.Log.Error(err)
			}
			explodedDNSRepo.Upsert(domainMap)
		} else {
			fmt.Println("\t[!] No DNS data to analyze")
		}

	}
}

//buildExplodedDNS .....
func (fs *FSImporter) buildCertificates(certMap map[string]*certificate.Input) {

	if len(certMap) > 0 {
		// Set up the database
		certificateRepo := certificate.NewMongoRepository(fs.res)
		err := certificateRepo.CreateIndexes()
		if err != nil {
			fs.res.Log.Error(err)
		}
		certificateRepo.Upsert(certMap)
	} else {
		fmt.Println("\t[!] No certificate data to analyze")
	}

}

//buildHostnames .....
func (fs *FSImporter) removeAnalysisChunk(cid int) error {

	// Set up the remover
	removerRepo := remover.NewMongoRemover(fs.res)
	err := removerRepo.Remove(cid)
	if err != nil {
		fs.res.Log.Error(err)
		return err
	}

	fs.res.MetaDB.SetChunk(cid, fs.res.DB.GetSelectedDB(), false)

	return nil

}

//buildHostnames .....
func (fs *FSImporter) buildHostnames(hostnameMap map[string]*hostname.Input) {
	// non-optional module
	if len(hostnameMap) > 0 {
		// Set up the database
		hostnameRepo := hostname.NewMongoRepository(fs.res)
		err := hostnameRepo.CreateIndexes()
		if err != nil {
			fs.res.Log.Error(err)
		}
		hostnameRepo.Upsert(hostnameMap)
	} else {
		fmt.Println("\t[!] No Hostname data to analyze")
	}

}

func (fs *FSImporter) buildUconns(uconnMap map[string]*uconn.Pair) {
	// non-optional module
	if len(uconnMap) > 0 {
		// Set up the database
		uconnRepo := uconn.NewMongoRepository(fs.res)

		err := uconnRepo.CreateIndexes()
		if err != nil {
			fs.res.Log.Error(err)
		}

		// send uconns to uconn analysis
		uconnRepo.Upsert(uconnMap)
	} else {
		fmt.Println("\t[!] No Uconn data to analyze")
		fmt.Printf("\t\t[!!] No local network traffic found, please check ")
		fmt.Println("InternalSubnets in your RITA config (/etc/rita/config.yaml)")
	}

}

func (fs *FSImporter) buildHosts(hostMap map[string]*host.IP) {
	// non-optional module
	if len(hostMap) > 0 {
		hostRepo := host.NewMongoRepository(fs.res)

		err := hostRepo.CreateIndexes()
		if err != nil {
			fs.res.Log.Error(err)
		}

		// send uconns to host analysis
		hostRepo.Upsert(hostMap)
	} else {
		fmt.Println("\t[!] No Host data to analyze")
		fmt.Printf("\t\t[!!] No local network traffic found, please check ")
		fmt.Println("InternalSubnets in your RITA config (/etc/rita/config.yaml)")
	}
}

func (fs *FSImporter) markBlacklistedPeers(hostMap map[string]*host.IP) {
	// non-optional module
	if len(hostMap) > 0 {
		blacklistRepo := blacklist.NewMongoRepository(fs.res)

		// send uconns to host analysis
		blacklistRepo.Upsert()
	}
}

func (fs *FSImporter) buildBeacons(uconnMap map[string]*uconn.Pair) {
	if fs.res.Config.S.Beacon.Enabled {
		if len(uconnMap) > 0 {
			beaconRepo := beacon.NewMongoRepository(fs.res)

			err := beaconRepo.CreateIndexes()
			if err != nil {
				fs.res.Log.Error(err)
			}

			// send uconns to beacon analysis
			beaconRepo.Upsert(uconnMap)
		} else {
			fmt.Println("\t[!] No Beacon data to analyze")
		}
	}

}

//buildUserAgent .....
func (fs *FSImporter) buildUserAgent(useragentMap map[string]*useragent.Input) {

	if fs.res.Config.S.UserAgent.Enabled {
		if len(useragentMap) > 0 {
			// Set up the database
			useragentRepo := useragent.NewMongoRepository(fs.res)
			err := useragentRepo.CreateIndexes()
			if err != nil {
				fs.res.Log.Error(err)
			}
			useragentRepo.Upsert(useragentMap)
		} else {
			fmt.Println("\t[!] No UserAgent data to analyze")
		}
	}
}

func (fs *FSImporter) updateTimestampRange() {
	session := fs.res.DB.Session.Copy()
	defer session.Close()

	// set collection name
	collectionName := fs.res.Config.T.Structure.UniqueConnTable

	// check if collection already exists
	names, _ := session.DB(fs.res.DB.GetSelectedDB()).CollectionNames()

	exists := false
	// make sure collection exists
	for _, name := range names {
		if name == collectionName {
			exists = true
			break
		}
	}

	if !exists {
		return
	}

	// Build query for aggregation
	timestampMinQuery := []bson.M{
		bson.M{"$project": bson.M{"_id": 0, "ts": "$dat.ts"}},
		bson.M{"$unwind": "$ts"},
		bson.M{"$unwind": "$ts"}, // Not an error, must unwind it twice
		bson.M{"$sort": bson.M{"ts": 1}},
		bson.M{"$limit": 1},
	}

	var resultMin struct {
		Timestamp int64 `bson:"ts"`
	}

	// get iminimum timestamp
	// sort by the timestamp, limit it to 1 (only returns first result)
	err := session.DB(fs.res.DB.GetSelectedDB()).C(collectionName).Pipe(timestampMinQuery).AllowDiskUse().One(&resultMin)

	if err != nil {
		fs.res.Log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not retrieve minimum timestamp:", err)
		return
	}

	// Build query for aggregation
	timestampMaxQuery := []bson.M{
		bson.M{"$project": bson.M{"_id": 0, "ts": "$dat.ts"}},
		bson.M{"$unwind": "$ts"},
		bson.M{"$unwind": "$ts"}, // Not an error, must unwind it twice
		bson.M{"$sort": bson.M{"ts": -1}},
		bson.M{"$limit": 1},
	}

	var resultMax struct {
		Timestamp int64 `bson:"ts"`
	}

	// get max timestamp
	// sort by the timestamp, limit it to 1 (only returns first result)
	err = session.DB(fs.res.DB.GetSelectedDB()).C(collectionName).Pipe(timestampMaxQuery).AllowDiskUse().One(&resultMax)

	if err != nil {
		fs.res.Log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not retrieve maximum timestamp:", err)
		return
	}

	// set range in metadatabase
	err = fs.res.MetaDB.AddTSRange(fs.res.DB.GetSelectedDB(), resultMin.Timestamp, resultMax.Timestamp)
	if err != nil {
		fs.res.Log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not set ts range in metadatabase: ", err)
	}

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

//isIPv4 checks if an ip is ipv4
func isIPv4(address string) bool {
	return strings.Count(address, ":") < 2
}

//ipv4ToBinary generates binary representations of the IPv4 addresses
func ipv4ToBinary(ipv4 net.IP) int64 {
	return int64(binary.BigEndian.Uint32(ipv4[12:16]))
}
