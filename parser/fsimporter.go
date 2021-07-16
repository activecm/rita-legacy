package parser

import (
	"fmt"
	"net"
	"os"
	"sort"
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
	"github.com/activecm/rita/pkg/explodeddns"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/pkg/remover"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/pkg/useragent"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"

	"github.com/globalsign/mgo/bson"
	"github.com/pbnjay/memory"
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
	// set batchSize to the max of 4GB or a half of system RAM to prevent running out of memory while importing
	batchSize := int64(util.MaxUint64(4*(1<<30), (memory.TotalMemory() / 2)))
	return &FSImporter{
		filter:         newFilter(res.Config),
		log:            res.Log,
		config:         res.Config,
		database:       res.DB,
		metaDB:         res.MetaDB,
		batchSizeBytes: batchSize,
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
		retVals := fs.parseFiles(indexedFileBatch, threads, fs.log)
		// Set chunk before we continue so if process dies, we still verify with a delete if
		// any data was written out.
		fs.metaDB.SetChunk(fs.config.S.Rolling.CurrentChunk, fs.database.GetSelectedDB(), true)

		// build Hosts table.
		fs.buildHosts(retVals.HostMap)

		// build Uconns table. Must go before beacons.
		fs.buildUconns(retVals.UniqueConnMap)

		// update ts range for dataset (needs to be run before beacons)
		minTimestamp, maxTimestamp := fs.updateTimestampRange()

		// build or update the exploded DNS table. Must go before hostnames
		fs.buildExplodedDNS(retVals.ExplodedDNSMap)

		// build or update the exploded DNS table
		fs.buildHostnames(retVals.HostnameMap)

		// build or update Beacons table
		fs.buildBeacons(retVals.UniqueConnMap, minTimestamp, maxTimestamp)

		// build or update the FQDN Beacons Table
		fs.buildFQDNBeacons(retVals.HostnameMap, minTimestamp, maxTimestamp)

		// build or update the Proxy Beacons Table
		fs.buildProxyBeacons(retVals.ProxyUniqueConnMap, minTimestamp, maxTimestamp)

		// build or update UserAgent table
		fs.buildUserAgent(retVals.UseragentMap)

		// build or update Certificate table
		fs.buildCertificates(retVals.CertificateMap)

		// update blacklisted peers in hosts collection
		fs.markBlacklistedPeers(retVals.HostMap)

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
func (fs *FSImporter) parseFiles(indexedFiles []*files.IndexedFile, parsingThreads int, logger *log.Logger) ParseResults {

	fmt.Println("\t[-] Parsing logs to: " + fs.database.GetSelectedDB() + " ... ")

	parseStartTime := time.Now()
	retVals := newParseResults()

	//set up parallel parsing
	n := len(indexedFiles)
	parsingWG := new(sync.WaitGroup)

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
				fileScanner, closeScanner, err := files.GetFileScanner(fileHandle)
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
					var entry parsetypes.BroData
					if indexedFiles[j].IsJSON() {
						entry = files.ParseJSONLine(fileScanner.Bytes(), indexedFiles[j].GetBroDataFactory(), logger)
					} else {
						// I've tried to increase performance by avoiding the allocations that result from
						// scanner.Text() by using .Bytes() with an unsafe cast, but that seemed to hurt performance -LL
						entry = files.ParseTSVLine(fileScanner.Text(),
							indexedFiles[j].GetHeader(), indexedFiles[j].GetFieldMap(),
							indexedFiles[j].GetBroDataFactory(), logger,
						)
					}

					if entry == nil {
						continue
					}

					switch typedEntry := entry.(type) {
					case *parsetypes.Conn:
						parseConnEntry(typedEntry, fs.filter, retVals)
					case *parsetypes.DNS:
						parseDNSEntry(typedEntry, fs.filter, retVals)
					case *parsetypes.HTTP:
						parseHTTPEntry(typedEntry, fs.filter, retVals)
					case *parsetypes.OpenConn:
						parseOpenConnEntry(typedEntry, fs.filter, retVals)
					case *parsetypes.SSL:
						parseSSLEntry(typedEntry, fs.filter, retVals)
					}
				}
				indexedFiles[j].ParseTime = time.Now()
				closeScanner() // handles closing the underlying fileHandle
				logger.WithFields(log.Fields{
					"path": indexedFiles[j].Path,
				}).Info("Finished parsing file")
			}
			wg.Done()
		}(indexedFiles, logger, parsingWG, i, parsingThreads, n)
	}
	parsingWG.Wait()
	fmt.Println("\t[-] Finished parsing logs in " + util.FormatDuration(
		time.Since(parseStartTime).Truncate(time.Millisecond)),
	)
	/*
		f, err := os.Create("./ram.pprof")
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	*/

	return retVals
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
