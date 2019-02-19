package parser

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	fpt "github.com/activecm/rita/parser/fileparsetypes"
	"github.com/activecm/rita/pkg/beacon"
	"github.com/activecm/rita/pkg/blacklist"
	"github.com/activecm/rita/pkg/explodeddns"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/hostname"
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

//Run starts importing a given path into a datastore
func (fs *FSImporter) Run(datastore Datastore) {
	// track the time spent parsing
	start := time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"start_time": start.Format(util.TimeFormat),
		},
	).Info("Starting filesystem import. Collecting file details.")

	fmt.Println("\t[-] Finding files to parse ... ")
	//find all of the bro log paths
	files := readDir(fs.res.Config.S.Bro.ImportDirectory, fs.res.Log)

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
		fmt.Println("\n\t[!] All files in this directory have already been parsed into database: ", fs.res.DB.GetSelectedDB())
		return
	}

	// create blacklisted reference Collection if blacklisted module is enabled
	if fs.res.Config.S.Blacklisted.Enabled {
		blacklist.BuildBlacklistedCollections(fs.res)
	}

	// parse in those files!
	uconnMap, explodeddnsMap, hostnameMap, useragentMap := fs.parseFiles(indexedFiles, fs.parseThreads, datastore, fs.res.Log)

	// Must wait for all mongodatastore inserts to finish
	datastore.Flush()

	// build Uconns table. Must go before beacons.
	fs.buildUconns(uconnMap)

	// build Hosts table.
	fs.buildHosts(uconnMap)

	// build or update the exploded DNS table. Must go before hostnames
	fs.buildExplodedDNS(explodeddnsMap)

	// build or update the exploded DNS table
	fs.buildHostnames(hostnameMap)

	// build or update Beacons table
	fs.buildBeacons(uconnMap)

	// build or update UserAgent table
	fs.buildUserAgent(useragentMap)

	// record file+database name hash in metadabase to prevent duplicate content
	fmt.Println("\t[-] Indexing log entries ... ")
	updateFilesIndex(indexedFiles, fs.res.MetaDB, fs.res.Log)

	// add min/max timestamps to metaDatabase and mark results as imported and analyzed
	fmt.Println("\t[-] Updating metadatabase ... ")
	fs.updateTimestampRange()
	fs.res.MetaDB.MarkDBImported(fs.res.DB.GetSelectedDB(), true)
	fs.res.MetaDB.MarkDBAnalyzed(fs.res.DB.GetSelectedDB(), true)

	progTime = time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"current_time": progTime.Format(util.TimeFormat),
			"total_time":   progTime.Sub(start).String(),
		},
	).Info("Finished upload. Starting indexing")

	datastore.Index()

	progTime = time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"current_time": progTime.Format(util.TimeFormat),
			"total_time":   progTime.Sub(start).String(),
		},
	).Info("Finished importing log files")

	fmt.Println("\t[-] Done!")
}

// readDir recursively reads the directory looking for log and .gz files
func readDir(cpath string, logger *log.Logger) []string {
	var toReturn []string
	files, err := ioutil.ReadDir(cpath)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
			"path":  cpath,
		}).Error("Error when reading directory")
	}

	for _, file := range files {
		// Stop RITA from following symlinks
		// In the case that RITA is pointed directly at Bro, it should not
		// parse the "current" symlink which points to the spool.
		if file.IsDir() && file.Mode() != os.ModeSymlink {
			toReturn = append(toReturn, readDir(path.Join(cpath, file.Name()), logger)...)
		}
		if strings.HasSuffix(file.Name(), "gz") ||
			strings.HasSuffix(file.Name(), "log") {
			toReturn = append(toReturn, path.Join(cpath, file.Name()))
		}
	}
	return toReturn
}

//indexFiles takes in a list of bro files, a number of threads, and parses
//some metadata out of the files
func indexFiles(files []string, indexingThreads int,
	cfg *config.Config, logger *log.Logger) []*fpt.IndexedFile {
	n := len(files)
	output := make([]*fpt.IndexedFile, n)
	indexingWG := new(sync.WaitGroup)

	for i := 0; i < indexingThreads; i++ {
		indexingWG.Add(1)

		go func(files []string, indexedFiles []*fpt.IndexedFile,
			sysConf *config.Config, logger *log.Logger,
			wg *sync.WaitGroup, start int, jump int, length int) {

			for j := start; j < length; j += jump {
				indexedFile, err := newIndexedFile(files[j], cfg, logger)
				if err != nil {
					logger.WithFields(log.Fields{
						"file":  files[j],
						"error": err.Error(),
					}).Warning("An error was encountered while indexing a file")
					//errored on files will be nil
					continue
				}
				indexedFiles[j] = indexedFile
			}
			wg.Done()
		}(files, output, cfg, logger, indexingWG, i, indexingThreads, n)
	}

	indexingWG.Wait()
	return output
}

//parseFiles takes in a list of indexed bro files, the number of
//threads to use to parse the files, whether or not to sort data by date,
//a MongoDB datastore object to store the bro data in, and a logger to report
//errors and parses the bro files line by line into the database.
func (fs *FSImporter) parseFiles(indexedFiles []*fpt.IndexedFile, parsingThreads int, datastore Datastore, logger *log.Logger) (
	map[string]*uconn.Pair, map[string]int, map[string][]string, map[string]*useragent.Input) {

	fmt.Println("\t[-] Parsing logs to: " + fs.res.DB.GetSelectedDB() + " ... ")
	// create log parsing maps
	explodeddnsMap := make(map[string]int)

	hostnameMap := make(map[string][]string)

	useragentMap := make(map[string]*useragent.Input)

	// Counts the number of uconns per source-destination pair
	uconnMap := make(map[string]*uconn.Pair)

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
				fmt.Printf("\r\t[-] Parsing " + indexedFiles[j].Path + " -> " + indexedFiles[j].TargetDatabase)

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

						/// *************************************************************///
						///                           CONNS                              ///
						/// *************************************************************///
						if targetCollection == fs.res.Config.T.Structure.ConnTable {

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
								bytes := int64(origIPBytes + respIPBytes)
								protocol := parseConn.FieldByName("Proto").Interface().(string)
								service := parseConn.FieldByName("Service").Interface().(string)
								dstPort := parseConn.FieldByName("DestinationPort").Interface().(int)

								// Concatenate the source and destination IPs to use as a map key
								srcDst := src + dst

								// Safely store the number of conns for this uconn
								mutex.Lock()

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
								}

								for _, entry := range trustedAppReferenceList {
									if (protocol == entry.protocol) && (dstPort == entry.port) {
										if service != entry.service {
											uconnMap[srcDst].UntrustedAppConnCount++
										}
									}
								}

								// Increment the connection count for the src-dst pair
								uconnMap[srcDst].ConnectionCount++

								// Only append unique timestamps to tslist
								if int64InSlice(ts, uconnMap[srcDst].TsList) == false {
									uconnMap[srcDst].TsList = append(uconnMap[srcDst].TsList, ts)
								}

								// Append all origIPBytes to origBytesList
								uconnMap[srcDst].OrigBytesList = append(uconnMap[srcDst].OrigBytesList, origIPBytes)

								// Calculate and store the total number of bytes exchanged by the uconn pair
								uconnMap[srcDst].TotalBytes += bytes

								// Calculate and store the total duration
								uconnMap[srcDst].TotalDuration += duration

								// Replace existing duration if current duration is higher
								if duration > uconnMap[srcDst].MaxDuration {
									uconnMap[srcDst].MaxDuration = duration
								}

								mutex.Unlock()

								// stores the conn record in conn collection if below threshold
								//
								// datastore.Store(&ImportedData{
								// 	BroData:          data,
								// 	TargetDatabase:   fs.res.DB.GetSelectedDB(),
								// 	TargetCollection: targetCollection,
								// })

							}

							/// *************************************************************///
							///                             DNS                             ///
							/// *************************************************************///
						} else if targetCollection == fs.res.Config.T.Structure.DNSTable {
							parseDNS := reflect.ValueOf(data).Elem()

							domain := parseDNS.FieldByName("Query").Interface().(string)
							queryTypeName := parseDNS.FieldByName("QTypeName").Interface().(string)

							// Safely store the number of conns for this uconn
							mutex.Lock()

							// increment domain map count for exploded dns
							explodeddnsMap[domain]++

							// Increment the connection count for the src-dst pair
							if _, ok := hostnameMap[domain]; !ok {
								hostnameMap[domain] = []string{}
							}

							if queryTypeName == "A" {
								answers := parseDNS.FieldByName("Answers").Interface().([]string)
								for _, answer := range answers {
									// make sure we aren't storing more than the configured max
									if len(hostnameMap[domain]) >= fs.res.Config.S.Hostname.IPListLimit {
										break
									}
									// Check if answer is an IP address
									if net.ParseIP(answer) != nil {
										if stringInSlice(answer, hostnameMap[domain]) == false {
											hostnameMap[domain] = append(hostnameMap[domain], answer)
										}

									}
								}
							}

							// increment txt query count for host in uconn
							if queryTypeName == "TXT" {
								// get source destination pair for dns record
								src := parseDNS.FieldByName("Source").Interface().(string)
								dst := parseDNS.FieldByName("Destination").Interface().(string)

								// Check if uconn map value is set, because this record could
								// come before a relevant conns record
								if _, ok := uconnMap[src+dst]; !ok {
									// create new uconn record with src and dst
									// Set IsLocalSrc and IsLocalDst fields based on InternalSubnets setting
									// we only need to do this once if the uconn record does not exist
									uconnMap[src+dst] = &uconn.Pair{
										Src:        src,
										Dst:        dst,
										IsLocalSrc: containsIP(fs.GetInternalSubnets(), net.ParseIP(src)),
										IsLocalDst: containsIP(fs.GetInternalSubnets(), net.ParseIP(dst)),
									}
								}
								// increment txt query count
								uconnMap[src+dst].TXTQueryCount++
							}

							mutex.Unlock()

							// stores the dns record in the dns collection
							// datastore.Store(&ImportedData{
							// 	BroData:          data,
							// 	TargetDatabase:   fs.res.DB.GetSelectedDB(),
							// 	TargetCollection: targetCollection,
							// })

							/// *************************************************************///
							///                             HTTP                             ///
							/// *************************************************************///
						} else if targetCollection == fs.res.Config.T.Structure.HTTPTable {
							parseHTTP := reflect.ValueOf(data).Elem()
							userAgentName := parseHTTP.FieldByName("UserAgent").Interface().(string)
							src := parseHTTP.FieldByName("Source").Interface().(string)
							host := parseHTTP.FieldByName("Host").Interface().(string)

							// Safely store the number of conns for this uconn
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

							// stores the http record in the http collection
							// datastore.Store(&ImportedData{
							// 	BroData:          data,
							// 	TargetDatabase:   fs.res.DB.GetSelectedDB(),
							// 	TargetCollection: targetCollection,
							// })

							/// *************************************************************///
							///                             SSL                             ///
							/// *************************************************************///
						} else if targetCollection == fs.res.Config.T.Structure.SSLTable {

							// parseSSL := reflect.ValueOf(data).Elem()

							// stores the ssl record in the ssl collection
							// datastore.Store(&ImportedData{
							// 	BroData:          data,
							// 	TargetDatabase:   fs.res.DB.GetSelectedDB(),
							// 	TargetCollection: targetCollection,
							// })

							/// *************************************************************///
							///                             x509                             ///
							/// *************************************************************///
						} else if targetCollection == fs.res.Config.T.Structure.X509Table {
							// datastore.Store(&ImportedData{
							// 	BroData:          data,
							// 	TargetDatabase:   fs.res.DB.GetSelectedDB(),
							// 	TargetCollection: targetCollection,
							// })

						} else {
							// We do not analyze any of the other log types (yet)
							// datastore.Store(&ImportedData{
							// 	BroData:          data,
							// 	TargetDatabase:   fs.res.DB.GetSelectedDB(),
							// 	TargetCollection: targetCollection,
							// })
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

	return uconnMap, explodeddnsMap, hostnameMap, useragentMap
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

//buildHostnames .....
func (fs *FSImporter) buildHostnames(hostnameMap map[string][]string) {
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

func (fs *FSImporter) buildHosts(uconnMap map[string]*uconn.Pair) {
	// non-optional module
	if len(uconnMap) > 0 {
		hostRepo := host.NewMongoRepository(fs.res)

		err := hostRepo.CreateIndexes()
		if err != nil {
			fs.res.Log.Error(err)
		}

		// send uconns to host analysis
		hostRepo.Upsert(uconnMap)
	} else {
		fmt.Println("\t[!] No Host data to analyze")
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
		bson.M{"$project": bson.M{"_id": 0, "dat": 1}},
		bson.M{"$unwind": "$dat"},
		bson.M{"$project": bson.M{"_id": 0, "ts": "$dat.ts"}},
		bson.M{"$unwind": "$ts"},
		bson.M{"$project": bson.M{"_id": 0, "ts": 1}},
		bson.M{"$sort": bson.M{"ts": 1}},
		bson.M{"$limit": 1},
	}

	var resultMin struct {
		Timestamp int64 `bson:"ts"`
	}

	// get iminimum timestamp
	// sort by the timestamp, limit it to 1 (only returns first result)
	err := session.DB(fs.res.DB.GetSelectedDB()).C(collectionName).Pipe(timestampMinQuery).One(&resultMin)

	if err != nil {
		fs.res.Log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not retrieve minimum timestamp:", err)
		return
	}

	// Build query for aggregation
	timestampMaxQuery := []bson.M{
		bson.M{"$project": bson.M{"_id": 0, "dat": 1}},
		bson.M{"$unwind": "$dat"},
		bson.M{"$project": bson.M{"_id": 0, "ts": "$dat.ts"}},
		bson.M{"$unwind": "$ts"},
		bson.M{"$project": bson.M{"_id": 0, "ts": 1}},
		bson.M{"$sort": bson.M{"ts": -1}},
		bson.M{"$limit": 1},
	}

	var resultMax struct {
		Timestamp int64 `bson:"ts"`
	}

	// get max timestamp
	// sort by the timestamp, limit it to 1 (only returns first result)
	err = session.DB(fs.res.DB.GetSelectedDB()).C(collectionName).Pipe(timestampMaxQuery).One(&resultMax)

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

//removeOldFilesFromIndex checks all indexedFiles passed in to ensure
//that they have not previously been imported into the same database.
//The files are compared based on their hashes (md5 of first 15000 bytes)
//and the database they are slated to be imported into.
func removeOldFilesFromIndex(indexedFiles []*fpt.IndexedFile,
	metaDatabase *database.MetaDB, logger *log.Logger) []*fpt.IndexedFile {
	var toReturn []*fpt.IndexedFile
	oldFiles, err := metaDatabase.GetFiles()
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not obtain a list of previously parsed files")
	}
	//NOTE: This can be improved to n log n if we need to
	for _, newFile := range indexedFiles {
		if newFile == nil {
			//this file was errored on earlier, i.e. we didn't find a tgtDB etc.
			continue
		}

		have := false
		for _, oldFile := range oldFiles {
			if oldFile.Hash == newFile.Hash && oldFile.TargetDatabase == newFile.TargetDatabase {
				logger.WithFields(log.Fields{
					"path":            newFile.Path,
					"target_database": newFile.TargetDatabase,
				}).Warning("Refusing to import file into the same database twice")
				have = true
				break
			}
		}

		if !have {
			toReturn = append(toReturn, newFile)
		}
	}
	return toReturn
}

//updateFilesIndex updates the files collection in the metaDB with the newly parsed files
func updateFilesIndex(indexedFiles []*fpt.IndexedFile, metaDatabase *database.MetaDB,
	logger *log.Logger) {
	err := metaDatabase.AddParsedFiles(indexedFiles)
	if err != nil {
		logger.Error("Could not update the list of parsed files")
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
