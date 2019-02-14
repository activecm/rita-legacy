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
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/pkg/beacon"
	"github.com/activecm/rita/pkg/blacklist"
	"github.com/activecm/rita/pkg/conn"
	"github.com/activecm/rita/pkg/explodeddns"
	"github.com/activecm/rita/pkg/freq"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/pkg/useragent"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
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
		connLimit:       int64(res.Config.S.Strobe.ConnectionLimit),
	}
}

//GetInternalSubnets returns the internal subnets from the config file
func (fs *FSImporter) GetInternalSubnets() []*net.IPNet {
	return fs.internal
}

//Run starts importing a given path into a datastore
func (fs *FSImporter) Run(datastore Datastore) {
	// build the rita-bl database before parsing
	if fs.res.Config.S.Blacklisted.Enabled {
		blacklist.BuildBlacklistedCollections(fs.res)
	}

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

	progTime := time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"current_time": progTime.Format(util.TimeFormat),
			"total_time":   progTime.Sub(start).String(),
		},
	).Info("Finished collecting file details. Starting upload.")

	indexedFiles = removeOldFilesFromIndex(indexedFiles, fs.res.MetaDB, fs.res.Log)

	// obviously this is temporary
	if !(len(indexedFiles) > 0) {
		fmt.Println("\n\t[!!!!!] dumb error with file hashing that ethan is working on fixing, please choose a different database name and try again! ")
		return
	}

	// create blacklisted reference Collection
	fmt.Println("\t[-] Creating blacklist reference collection ... ")
	blacklist.BuildBlacklistedCollections(fs.res)

	// parse in those files!
	filterHugeUconnsMap, uconnMap, explodeddnsMap, hostnameMap, useragentMap := fs.parseFiles(indexedFiles, fs.parseThreads, datastore, fs.res.Log)

	// Must wait for all mongodatastore inserts to finish before attempting to delete
	datastore.Flush()
	fs.bulkRemoveHugeUconns(filterHugeUconnsMap, uconnMap)

	// build Uconns table
	fs.buildUconns(uconnMap)

	// build Hosts table
	fs.buildHosts(uconnMap)

	// build or update the exploded DNS table
	fs.buildExplodedDNS(explodeddnsMap)

	// build or update the exploded DNS table
	fs.buildHostnames(hostnameMap)

	// build or update Beacons table
	fs.buildBeacons(uconnMap)

	// build or update UserAgent table
	fs.buildUserAgent(useragentMap)

	fmt.Println("\t[-] Waiting for all inserts to finish ... ")

	fmt.Println("\t[-] Indexing log entries ... ")
	updateFilesIndex(indexedFiles, fs.res.MetaDB, fs.res.Log)

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
	[]uconn.Pair, map[string]uconn.Pair, map[string]int, map[string][]string, map[string]*useragent.Input) {

	fmt.Println("\t[-] Parsing logs to: " + fs.res.DB.GetSelectedDB() + " ... ")
	// create log parsing maps
	explodeddnsMap := make(map[string]int)

	hostnameMap := make(map[string][]string)

	useragentMap := make(map[string]*useragent.Input)

	// Counts the number of uconns per source-destination pair
	uconnMap := make(map[string]uconn.Pair)

	//set up parallel parsing
	n := len(indexedFiles)
	parsingWG := new(sync.WaitGroup)

	// map to hold the too many connections uconns
	var filterHugeUconnsMap []uconn.Pair

	// Creates a mutex for locking map keys during read-write operations
	var mutex = &sync.Mutex{}

	for i := 0; i < parsingThreads; i++ {
		parsingWG.Add(1)

		go func(indexedFiles []*fpt.IndexedFile, logger *log.Logger,
			wg *sync.WaitGroup, start int, jump int, length int) {
			//comb over array
			for j := start; j < length; j += jump {
				fmt.Println("\t[-] Parsing " + indexedFiles[j].Path + " -> " + indexedFiles[j].TargetDatabase)

				//read the file
				fileHandle, err := os.Open(indexedFiles[j].Path)
				if err != nil {
					logger.WithFields(log.Fields{
						"file":  indexedFiles[j].Path,
						"error": err.Error(),
					}).Error("Could not open file for parsing")
				}
				fileScanner, err := getFileScanner(fileHandle)
				if err != nil {
					logger.WithFields(log.Fields{
						"file":  indexedFiles[j].Path,
						"error": err.Error(),
					}).Error("Could not open file for parsing")
				}

				for fileScanner.Scan() {
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
							uconnPair := uconn.Pair{
								Src: parseConn.FieldByName("Source").Interface().(string),
								Dst: parseConn.FieldByName("Destination").Interface().(string),
							}

							// Run conn pair through filter to filter out certain connections
							ignore := fs.filterConnPair(uconnPair.Src, uconnPair.Dst)

							// If connection pair is not subject to filtering, process
							if !ignore {
								// Set IsLocalSrc and IsLocalDst fields based on InternalSubnets setting
								uconnPair.IsLocalSrc = containsIP(fs.GetInternalSubnets(), net.ParseIP(uconnPair.Src))
								uconnPair.IsLocalDst = containsIP(fs.GetInternalSubnets(), net.ParseIP(uconnPair.Dst))
								ts := parseConn.FieldByName("TimeStamp").Interface().(int64)
								origIPBytes := parseConn.FieldByName("OrigIPBytes").Interface().(int64)
								respIPBytes := parseConn.FieldByName("RespIPBytes").Interface().(int64)
								duration := float64(parseConn.FieldByName("Duration").Interface().(float64))
								bytes := int64(origIPBytes + respIPBytes)

								// Concatenate the source and destination IPs to use as a map key
								srcDst := uconnPair.Src + uconnPair.Dst

								// Safely store the number of conns for this uconn
								mutex.Lock()

								// Increment the connection count for the src-dst pair
								connCount := uconnMap[srcDst].ConnectionCount + 1
								uconnPair.ConnectionCount = connCount

								// Only append unique timestamps to tslist
								timestamps := uconnMap[srcDst].TsList
								if isUniqueTimestamp(ts, timestamps) {
									uconnPair.TsList = append(timestamps, ts)
								} else {
									uconnPair.TsList = timestamps
								}

								// Append all origIPBytes to origBytesList
								uconnPair.OrigBytesList = append(uconnMap[srcDst].OrigBytesList, origIPBytes)

								// Calculate and store the total number of bytes exchanged by the uconn pair
								uconnPair.TotalBytes = uconnMap[srcDst].TotalBytes + bytes

								// Calculate and store the average number of bytes
								uconnPair.AvgBytes = float64(((int64(uconnMap[srcDst].AvgBytes) * connCount) + bytes) / (connCount + 1))

								// Calculate and store the total duration
								uconnPair.TotalDuration = uconnMap[srcDst].TotalDuration + duration

								// Replace existing duration if current duration is higher
								if duration > uconnMap[srcDst].MaxDuration {
									uconnPair.MaxDuration = duration
								} else {
									uconnPair.MaxDuration = uconnMap[srcDst].MaxDuration
								}
								uconnMap[srcDst] = uconn.Pair{
									Src:             uconnPair.Src,
									Dst:             uconnPair.Dst,
									ConnectionCount: uconnPair.ConnectionCount,
									IsLocalSrc:      uconnPair.IsLocalSrc,
									IsLocalDst:      uconnPair.IsLocalDst,
									TotalBytes:      uconnPair.TotalBytes,
									AvgBytes:        uconnPair.AvgBytes,
									TotalDuration:   uconnPair.TotalDuration,
									MaxDuration:     uconnPair.MaxDuration,
									TsList:          uconnPair.TsList,
									OrigBytesList:   uconnPair.OrigBytesList,
								}

								if connCount == fs.connLimit {
									// tag strobe for removal from conns after import
									filterHugeUconnsMap = append(filterHugeUconnsMap, uconnPair)
								}

								mutex.Unlock()

								// stores the conn record in conn collection if below threshold
								if connCount < fs.connLimit {
									datastore.Store(&ImportedData{
										BroData:          data,
										TargetDatabase:   fs.res.DB.GetSelectedDB(),
										TargetCollection: targetCollection,
									})
								}
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
								useragentMap[userAgentName] = &useragent.Input{OrigIps: []string{src}, Seen: 1, Requests: []string{host}}
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

	return filterHugeUconnsMap, uconnMap, explodeddnsMap, hostnameMap, useragentMap
}

func isUniqueTimestamp(timestamp int64, timestamps []int64) bool {
	for _, val := range timestamps {
		if val == timestamp {
			return false
		}
	}
	return true
}

//buildExplodedDNS .....
func (fs *FSImporter) buildExplodedDNS(domainMap map[string]int) {
	fmt.Println("\t[-] Creating Exploded DNS Collection ...")
	// Set up the database
	explodedDNSRepo := explodeddns.NewMongoRepository(fs.res)
	explodedDNSRepo.CreateIndexes()
	explodedDNSRepo.Upsert(domainMap)
}

//buildHostnames .....
func (fs *FSImporter) buildHostnames(hostnameMap map[string][]string) {
	fmt.Println("\t[-] Creating Hostnames Collection ...")
	// Set up the database
	hostnameRepo := hostname.NewMongoRepository(fs.res)
	hostnameRepo.CreateIndexes()
	hostnameRepo.Upsert(hostnameMap)
}

func (fs *FSImporter) buildUconns(uconnMap map[string]uconn.Pair) {
	fmt.Println("\t[-] Creating Uconns Collection ...")

	uconnRepo := uconn.NewMongoRepository(fs.res)

	err := uconnRepo.CreateIndexes()
	if err != nil {
		fs.res.Log.Error(err)
	}

	// send uconns to uconn analysis
	uconnRepo.Upsert(uconnMap)

}

func (fs *FSImporter) buildHosts(uconnMap map[string]uconn.Pair) {
	fmt.Println("\t[-] Creating Hosts Collection ...")
	hostRepo := host.NewMongoRepository(fs.res)

	err := hostRepo.CreateIndexes()
	if err != nil {
		fs.res.Log.Error(err)
	}

	// send uconns to host analysis
	hostRepo.Upsert(uconnMap)
}

func (fs *FSImporter) buildBeacons(uconnMap map[string]uconn.Pair) {
	fmt.Println("\t[-] Creating Beacons Collection ...")

	beaconRepo := beacon.NewMongoRepository(fs.res)

	err := beaconRepo.CreateIndexes()
	if err != nil {
		fs.res.Log.Error(err)
	}

	// send uconns to beacon analysis
	beaconRepo.Upsert(uconnMap)

}

//buildUserAgent .....
func (fs *FSImporter) buildUserAgent(useragentMap map[string]*useragent.Input) {
	fmt.Println("\t[-] Creating UserAgent Collection ...")
	// Set up the database
	useragentRepo := useragent.NewMongoRepository(fs.res)
	useragentRepo.CreateIndexes()
	useragentRepo.Upsert(useragentMap)
}

// bulkRemoveHugeUconns loops through every IP pair in filterHugeUconnsMap and deletes all corresponding
// entries in the "conn" collection. It also creates new entries in the FrequentConnTable collection.
func (fs *FSImporter) bulkRemoveHugeUconns(filterHugeUconnsMap []uconn.Pair, uconnMap map[string]uconn.Pair) {

	connRepo := conn.NewMongoRepository(fs.res)
	freqRepo := freq.NewMongoRepository(fs.res)

	fmt.Println("\t[-] Creating Strobes and removing unused connection info ... ")
	freqConns := make([]*parsetypes.Conn, 0)
	for _, freqConn := range filterHugeUconnsMap {
		freqConns = append(freqConns, &parsetypes.Conn{
			Source:      freqConn.Src,
			Destination: freqConn.Dst,
		})
		freqRepo.Insert(
			&parsetypes.Freq{
				Source:          freqConn.Src,
				Destination:     freqConn.Dst,
				ConnectionCount: freqConn.ConnectionCount,
			})
		// remove entry out of uconns map so it doesn't end up in uconns collection
		srcDst := freqConn.Src + freqConn.Dst
		delete(uconnMap, srcDst)
	}

	// Execute the bulk deletion
	connRepo.BulkDelete(freqConns)
}

//removeOldFilesFromIndex checks all indexedFiles passed in to ensure
//that they have not previously been imported into the same database.
//The files are compared based on their hashes (md5 of first 15000 bytes)
//and the database they are slated to be imported into.
func removeOldFilesFromIndex(indexedFiles []*fpt.IndexedFile,
	metaDatabase *database.MetaDB, logger *log.Logger) []*fpt.IndexedFile {
	var toReturn []*fpt.IndexedFile
	// oldFiles, err := metaDatabase.GetFiles()
	// if err != nil {
	// 	logger.WithFields(log.Fields{
	// 		"error": err.Error(),
	// 	}).Error("Could not obtain a list of previously parsed files")
	// }
	//NOTE: This can be improved to n log n if we need to
	for _, newFile := range indexedFiles {
		if newFile == nil {
			//this file was errored on earlier, i.e. we didn't find a tgtDB etc.
			continue
		}

		have := false
		// for _, oldFile := range oldFiles {
		// 	if oldFile.Hash == newFile.Hash && oldFile.TargetDatabase == newFile.TargetDatabase {
		// 		logger.WithFields(log.Fields{
		// 			"path":            newFile.Path,
		// 			"target_database": newFile.TargetDatabase,
		// 		}).Warning("Refusing to import file into the same database twice")
		// 		have = true
		// 		break
		// 	}
		// }

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
