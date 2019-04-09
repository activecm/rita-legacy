package parser

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	fpt "github.com/activecm/rita/parser/fileparsetypes"
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
		rolling         bool
		totalChunks     int
		currentChunk    int
		indexingThreads int
		parseThreads    int
		internal        []*net.IPNet
		alwaysIncluded  []*net.IPNet
		neverIncluded   []*net.IPNet
		connLimit       int64
	}

	trustedAppTiplet struct {
		protocol string
		port     int
		service  string
	}
)

//NewFSImporter creates a new file system importer
func NewFSImporter(res *resources.Resources,
	indexingThreads int, parseThreads int) *FSImporter {
	return &FSImporter{
		res:             res,
		rolling:         res.Config.S.Bro.Rolling,
		totalChunks:     res.Config.S.Bro.TotalChunks,
		currentChunk:    res.Config.S.Bro.CurrentChunk,
		indexingThreads: indexingThreads,
		parseThreads:    parseThreads,
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

//Run starts the importing
func (fs *FSImporter) Run() {
	// track the time spent parsing
	start := time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"start_time": start.Format(util.TimeFormat),
		},
	).Info("Starting filesystem import. Collecting file details.")

	var files []string
	//find all of the potential bro log paths

	// if rolling dataset
	if fs.rolling {

		fmt.Println("\t[-] Finding next chunk's files to parse ... ")
		files = readDirRolling(fs.currentChunk, fs.totalChunks, fs.res.Config.S.Bro.ImportDirectory, fs.res.Log)

	} else { // if regular dataset
		fmt.Println("\t[-] Finding files to parse ... ")
		files = readDir(fs.res.Config.S.Bro.ImportDirectory, fs.res.Log)

	}

	//hash the files and get their stats
	indexedFiles := indexFiles(files, fs.indexingThreads, fs.res.Config, fs.res.Log)

	// if no compatible files for import were found, handle error
	if !(len(indexedFiles) > 0) {
		fmt.Println("\n\t[!] No compatible log files found in directory")
		return
	}

	progTime := time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"current_time": progTime.Format(util.TimeFormat),
			"total_time":   progTime.Sub(start).String(),
		},
	).Info("Finished collecting file details. Starting upload.")

	fmt.Println("\t[-] Verifying log files have not been previously parsed into the target dataset ... ")
	// check list of files against metadatabase records to ensure that the a file
	// won't be imported into the same database twice.
	indexedFiles = removeOldFilesFromIndex(indexedFiles, fs.res.MetaDB, fs.res.Log)

	// if all files were removed because they've already been imported, handle error
	if !(len(indexedFiles) > 0) {
		if fs.rolling {
			fmt.Println("\t[!] All files pertaining to the current chunk entry have already been parsed into database: ", fs.res.DB.GetSelectedDB())
		} else {
			fmt.Println("\t[!] All files in this directory have already been parsed into database: ", fs.res.DB.GetSelectedDB())
		}
		return
	}

	if fs.rolling {
		chunkSet, err := fs.res.MetaDB.IsChunkSet(fs.currentChunk, fs.res.DB.GetSelectedDB())
		if err != nil {
			fmt.Println("\t[!] Could not find CID List entry in metadatabase")
			return
		}

		if chunkSet {
			fmt.Println("\t[-] Removing outdated data from rolling dataset ... ")
			fs.removeAnalysisChunk(fs.currentChunk)
		}
	}

	// create blacklisted reference Collection if blacklisted module is enabled
	if fs.res.Config.S.Blacklisted.Enabled {
		blacklist.BuildBlacklistedCollections(fs.res)
	}

	// parse in those files!
	uconnMap, hostMap, explodeddnsMap, hostnameMap, useragentMap, certMap := fs.parseFiles(indexedFiles, fs.parseThreads, fs.res.Log)

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
	updateFilesIndex(indexedFiles, fs.res.MetaDB, fs.res.Log)

	// add min/max timestamps to metaDatabase and mark results as imported and analyzed
	fmt.Println("\t[-] Updating metadatabase ... ")
	fs.res.MetaDB.MarkDBAnalyzed(fs.res.DB.GetSelectedDB(), true)

	progTime = time.Now()
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

//parseFiles takes in a list of indexed bro files, the number of
//threads to use to parse the files, whether or not to sort data by date,
//a MongoDB datastore object to store the bro data in, and a logger to report
//errors and parses the bro files line by line into the database.
func (fs *FSImporter) parseFiles(indexedFiles []*fpt.IndexedFile, parsingThreads int, logger *log.Logger) (
	map[string]*uconn.Pair, map[string]*host.IP, map[string]int, map[string]*hostname.Input, map[string]*useragent.Input, map[string]*certificate.Input) {

	fmt.Println("\t[-] Parsing logs to: " + fs.res.DB.GetSelectedDB() + " ... ")
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

							// Use reflection to access the conn entry's fields. At this point inside
							// the if statement we know parseConn is a "conn" instance, but the code
							// assumes a generic "BroType" interface.
							parseConn := reflect.ValueOf(data).Elem()

							// get source destination pair for connection record
							src := parseConn.FieldByName("Source").Interface().(string)
							dst := parseConn.FieldByName("Destination").Interface().(string)

							// Run conn pair through filter to filter out certain connections
							ignore := fs.filterConnPair(src, dst)

							// If connection pair is not subject to filtering, process
							if !ignore {
								ts := parseConn.FieldByName("TimeStamp").Interface().(int64)
								origIPBytes := parseConn.FieldByName("OrigIPBytes").Interface().(int64)
								respIPBytes := parseConn.FieldByName("RespIPBytes").Interface().(int64)
								duration := float64(parseConn.FieldByName("Duration").Interface().(float64))
								duration = math.Ceil((duration)*10000) / 10000
								bytes := int64(origIPBytes + respIPBytes)
								protocol := parseConn.FieldByName("Proto").Interface().(string)
								service := parseConn.FieldByName("Service").Interface().(string)
								dstPort := parseConn.FieldByName("DestinationPort").Interface().(int)
								var tuple string
								if service == "" {
									tuple = strconv.Itoa(dstPort) + ":" + protocol + ":-"
								} else {
									tuple = strconv.Itoa(dstPort) + ":" + protocol + ":" + service
								}

								// Concatenate the source and destination IPs to use as a map key
								srcDst := src + dst

								// Safely store the number of conns for this uconn
								mutex.Lock()

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

								// increment unique dst port: proto : sevice tuple list for host
								if stringInSlice(tuple, uconnMap[srcDst].Tuples) == false {
									uconnMap[srcDst].Tuples = append(uconnMap[srcDst].Tuples, tuple)
								}

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

								if stringInSlice(dst, hostMap[src].ConnectedDstHosts) == false {
									hostMap[src].ConnectedDstHosts = append(hostMap[src].ConnectedDstHosts, dst)
								}

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
							parseDNS := reflect.ValueOf(data).Elem()

							domain := parseDNS.FieldByName("Query").Interface().(string)
							queryTypeName := parseDNS.FieldByName("QTypeName").Interface().(string)

							// Safely store the number of conns for this uconn
							mutex.Lock()

							// increment domain map count for exploded dns
							explodeddnsMap[domain]++

							// initialize the hostname input objects for new hostnames
							if _, ok := hostnameMap[domain]; !ok {
								hostnameMap[domain] = &hostname.Input{}
							}

							// geo.vortex.data.microsoft.com.akadns.net

							// extract and store the dns client ip address
							src := parseDNS.FieldByName("Source").Interface().(string)
							if stringInSlice(src, hostnameMap[domain].ClientIPs) == false {
								hostnameMap[domain].ClientIPs = append(hostnameMap[domain].ClientIPs, src)
							}

							if queryTypeName == "A" {
								answers := parseDNS.FieldByName("Answers").Interface().([]string)
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
								dst := parseDNS.FieldByName("Destination").Interface().(string)

								// Run conn pair through filter to filter out certain connections
								ignore := fs.filterConnPair(src, dst)
								if !ignore {
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
							parseHTTP := reflect.ValueOf(data).Elem()
							userAgentName := parseHTTP.FieldByName("UserAgent").Interface().(string)
							src := parseHTTP.FieldByName("Source").Interface().(string)
							host := parseHTTP.FieldByName("Host").Interface().(string)

							// Safely store useragent information
							mutex.Lock()

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
							parseSSL := reflect.ValueOf(data).Elem()
							ja3Hash := parseSSL.FieldByName("JA3").Interface().(string)
							src := parseSSL.FieldByName("Source").Interface().(string)
							dst := parseSSL.FieldByName("Destination").Interface().(string)
							host := parseSSL.FieldByName("ServerName").Interface().(string)
							certStatus := parseSSL.FieldByName("ValidationStatus").Interface().(string)

							// fmt.Println(ja3Hash)
							// Safely store ja3 information
							mutex.Lock()

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
									// fmt.Println(certStatus)
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

									// update relevant cert record
									if _, ok := certMap[dst]; !ok {
										// create new uconn record if it does not exist
										certMap[dst] = &certificate.Input{
											Host: dst,
										}
									}

									// increment times seen count
									certMap[dst].Seen++

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
		fmt.Println(err)
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
