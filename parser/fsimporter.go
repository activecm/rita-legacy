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
	}

	uconnPair struct {
		id              int
		src             string
		dst             string
		connectionCount int64
		isLocalSrc      bool
		isLocalDst      bool
		totalBytes      int64
		avgBytes        float64
		maxDuration     float64
		tsList          []int64
		origBytesList   []int64
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

//Run starts importing a given path into a datastore
func (fs *FSImporter) Run(datastore Datastore) {
	// track the time spent parsing
	start := time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"start_time": start.Format(util.TimeFormat),
		},
	).Info("Starting filesystem import. Collecting file details.")

	fmt.Println("\t[-] Finding files to parse")
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

	filterHugeUconnsMap, uconnMap := fs.parseFiles(indexedFiles, fs.parseThreads, datastore, fs.res.Log)

	// Must wait for all inserts to finish before attempting to delete
	datastore.Flush()
	fs.bulkRemoveHugeUconns(indexedFiles[0].TargetDatabase, filterHugeUconnsMap, uconnMap)

	updateFilesIndex(indexedFiles, fs.res.MetaDB, fs.res.Log)

	progTime = time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"current_time": progTime.Format(util.TimeFormat),
			"total_time":   progTime.Sub(start).String(),
		},
	).Info("Finished upload. Starting indexing")
	fmt.Println("\t[-] Indexing log entries. This may take a while.")
	datastore.Index()

	progTime = time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"current_time": progTime.Format(util.TimeFormat),
			"total_time":   progTime.Sub(start).String(),
		},
	).Info("Finished importing log files")
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
func (fs *FSImporter) parseFiles(indexedFiles []*fpt.IndexedFile, parsingThreads int, datastore Datastore, logger *log.Logger) ([]uconnPair, map[string]uconnPair) {

	//set up parallel parsing
	n := len(indexedFiles)
	parsingWG := new(sync.WaitGroup)

	// Counts the number of uconns per source-destination pair
	uconnMap := make(map[string]uconnPair)

	// map to hold the too many connections uconns
	var filterHugeUconnsMap []uconnPair

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
					// The number of conns in a uconn
					var connCount int64 = 0
					// The maximum number of conns that will be stored
					// We need to move this somewhere where the importer & analyzer can both access it
					var connLimit int64 = int64(fs.res.Config.S.Strobe.ConnectionLimit)

					if data != nil {
						//figure out what database this line is heading for
						targetCollection := indexedFiles[j].TargetCollection
						targetDB := indexedFiles[j].TargetDatabase

						// if target collection is the conns table we want to limit
						// conns entries to unique connection pairs with fewer than connLimit
						// records
						if targetCollection == fs.res.Config.T.Structure.ConnTable {
							parseConn := reflect.ValueOf(data).Elem()

							var uconn uconnPair

							// Use reflection to access the conn entry's fields. At this point inside
							// the if statement we know parseConn is a "conn" instance, but the code
							// assumes a generic "BroType" interface.
							uconn.id = 0
							uconn.src = parseConn.FieldByName("Source").Interface().(string)
							uconn.dst = parseConn.FieldByName("Destination").Interface().(string)
							uconn.isLocalSrc = parseConn.FieldByName("LocalOrigin").Interface().(bool)
							uconn.isLocalDst = parseConn.FieldByName("LocalResponse").Interface().(bool)

							ts := parseConn.FieldByName("TimeStamp").Interface().(int64)
							origIPBytes := parseConn.FieldByName("OrigIPBytes").Interface().(int64)
							respIPBytes := parseConn.FieldByName("RespIPBytes").Interface().(int64)

							duration := float64(parseConn.FieldByName("Duration").Interface().(float64))

							bytes := int64(origIPBytes + respIPBytes)

							// Concatenate the source and destination IPs to use as a map key
							srcDst := uconn.src + uconn.dst

							// Run conn pair through filter to filter out certain connections
							ignore := false //fs.filterConnPair(uconn.src, uconn.dst)

							// If connection pair is not subject to filtering, process
							if !ignore {
								// Safely store the number of conns for this uconn
								mutex.Lock()

								// Increment the connection count for the src-dst pair
								connCount = uconnMap[srcDst].connectionCount + 1
								uconn.connectionCount = connCount

								// Append all timestamps to tsList
								uconn.tsList = append(uconnMap[srcDst].tsList, ts)

								// Append all origIPBytes to origBytesList
								uconn.origBytesList = append(uconnMap[srcDst].origBytesList, origIPBytes)

								// Calculate and store the total number of bytes exchanged by the uconn pair
								uconn.totalBytes = uconnMap[srcDst].totalBytes + bytes

								// Calculate and store the average number of bytes
								uconn.avgBytes = float64(((int64(uconnMap[srcDst].avgBytes) * connCount) + bytes) / (connCount + 1))

								// Replace existing duration if current duration is higher
								if duration > uconnMap[srcDst].maxDuration {
									uconn.maxDuration = duration
								} else {
									uconn.maxDuration = uconnMap[srcDst].maxDuration
								}

								uconnMap[srcDst] = uconnPair{
									id:              uconn.id,
	    						src:             uconn.src,
	    						dst:             uconn.dst,
	    						connectionCount: uconn.connectionCount,
	    						isLocalSrc:      uconn.isLocalSrc,
	    						isLocalDst:      uconn.isLocalDst,
    							totalBytes:      uconn.totalBytes,
    							avgBytes:        uconn.avgBytes,
    							maxDuration:     uconn.maxDuration,
    							tsList:          uconn.tsList,
    							origBytesList:   uconn.origBytesList,
								}

								// Do not store more than the connLimit
								if connCount < connLimit {
									datastore.Store(&ImportedData{
										BroData:          data,
										TargetDatabase:   targetDB,
										TargetCollection: targetCollection,
									})
								} else if connCount == connLimit {
									// Once we know a uconn has passed the connLimit not only
									// do we want to avoid storing any more, but we want to
									// remove all entries already added. The first time we pass
									// the limit put an entry in filterHugeUconnsMap in order
									// to run a bulk delete later.
									filterHugeUconnsMap = append(filterHugeUconnsMap, uconn)
								}

								mutex.Unlock()
							}
						} else {
							// We do not limit any of the other log types
							datastore.Store(&ImportedData{
								BroData:          data,
								TargetDatabase:   targetDB,
								TargetCollection: targetCollection,
							})
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

	return filterHugeUconnsMap, uconnMap
}

// bulkRemoveHugeUconns loops through every IP pair in filterHugeUconnsMap and deletes all corresponding
// entries in the "conn" collection. It also creates new entries in the FrequentConnTable collection.
func (fs *FSImporter) bulkRemoveHugeUconns(targetDB string, filterHugeUconnsMap []uconnPair, uconnMap map[string]uconnPair) {
	resDB := fs.res.DB
	resConf := fs.res.Config
	logger := fs.res.Log
	var deleteQuery bson.M

	// create a new datastore just for frequent connections since the old datastore
	// will be Flushed by now and closed for writing
	datastore := NewMongoDatastore(resDB.Session, fs.res.MetaDB,
		resConf.S.Bro.ImportBuffer, logger)

	// open a new database session for the bulk deletion
	ssn := resDB.Session.Copy()
	defer ssn.Close()

	bulk := ssn.DB(targetDB).C(resConf.T.Structure.ConnTable).Bulk()
	bulk.Unordered()

	fmt.Println("\t[-] Removing unused connection info. This may take a while.")
	for _, uconn := range filterHugeUconnsMap {
		datastore.Store(&ImportedData{
			BroData: &parsetypes.Freq{
				Source:          uconn.src,
				Destination:     uconn.dst,
				ConnectionCount: uconn.connectionCount,
			},
			TargetDatabase:   targetDB,
			TargetCollection: resConf.T.Structure.FrequentConnTable,
		})

		deleteQuery = bson.M{
			"$and": []bson.M{
				bson.M{"id_orig_h": uconn.src},
				bson.M{"id_resp_h": uconn.dst},
			}}
		bulk.RemoveAll(deleteQuery)

		// remove entry out of uconns map so it doesn't end up in uconns collection
		srcDst := uconn.src + uconn.dst
		delete(uconnMap, srcDst)
	}

	// lets create some uconns
	for uconn := range uconnMap {
		datastore.Store(&ImportedData{
			BroData: &parsetypes.Uconn{
				Source:           uconnMap[uconn].src,
				Destination:      uconnMap[uconn].dst,
				ConnectionCount:  uconnMap[uconn].connectionCount,
				LocalSource:      uconnMap[uconn].isLocalSrc,
				LocalDestination: uconnMap[uconn].isLocalDst,
				TotalBytes:       uconnMap[uconn].totalBytes,
				AverageBytes:     uconnMap[uconn].avgBytes,
				MaxDuration:      uconnMap[uconn].maxDuration,
				TSList:           uconnMap[uconn].tsList,
				OrigBytesList:    uconnMap[uconn].origBytesList,
			},
			TargetDatabase:   targetDB,
			TargetCollection: resConf.T.Structure.UniqueConnTable,
		})

	}

	// Flush the datastore to ensure that it finishes all of its writes
	datastore.Flush()
	datastore.Index()

	// Execute the bulk deletion
	bulkResult, err := bulk.Run()
	if err != nil {
		logger.WithFields(log.Fields{
			"bulkResult": bulkResult,
			"error":      err.Error(),
		}).Error("Could not delete frequent conn entries.")
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
