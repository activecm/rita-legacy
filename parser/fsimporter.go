package parser

import (
	"fmt"
	"io/ioutil"
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
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	//FSImporter provides the ability to import bro files from the file system
	FSImporter struct {
		res             *resources.Resources
		indexingThreads int
		parseThreads    int
	}

	uconnPair struct {
		src string
		dst string
	}
)

//NewFSImporter creates a new file system importer
func NewFSImporter(resources *resources.Resources,
	indexingThreads int, parseThreads int) *FSImporter {
	return &FSImporter{
		res:             resources,
		indexingThreads: indexingThreads,
		parseThreads:    parseThreads,
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

	fs.parseFiles(indexedFiles, fs.parseThreads, datastore, fs.res.Log)

	datastore.Flush()
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
// a MogoDB datastore object to store the bro data in, and a logger to report
//errors and parses the bro files line by line into the database.
func (fs *FSImporter) parseFiles(indexedFiles []*fpt.IndexedFile, parsingThreads int, datastore Datastore, logger *log.Logger) {
	// var connMap = make(map[uconnPair]int)

	//set up parallel parsing
	n := len(indexedFiles)
	parsingWG := new(sync.WaitGroup)

	// Counts the number of uconns per source-destination pair
	connMap := make(map[uconnPair]int)

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
					connCount := 0
					// The maximum number of conns that will be stored
					// We need to move this somewhere where the importer & analyzer can both access it
					connLimit := 250000

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

							uconn.src = parseConn.Field(3).Interface().(string)
							uconn.dst = parseConn.Field(5).Interface().(string)

							// Safely store the number of conns for this uconn
							mutex.Lock()
							connMap[uconn] = connMap[uconn] + 1
							connCount = connMap[uconn]

							// Do not store more than the connLimit
							if connCount < connLimit {
								datastore.Store(&ImportedData{
									BroData:          data,
									TargetDatabase:   targetDB,
									TargetCollection: targetCollection,
								})
							} else if connCount == connLimit {

								filterHugeUconnsMap = append(filterHugeUconnsMap, uconn)
								// fmt.Println(uconn.src, uconn.dst, connCount)
								// datastore.Store(&ImportedData{
								// 	BroData:          data,
								// 	TargetDatabase:   targetDB,
								// 	TargetCollection: "temp",
								// })
							}

							mutex.Unlock()

						} else {

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

	fs.bulkRemoveHugeUconns(datastore, indexedFiles[0].TargetDatabase, filterHugeUconnsMap, connMap)
}

// robomongo verification stuf:
// pre-bulk delete conns should have exactly [connLimit - 1] number of records
// of each src dst pair entry found in new temp collection:
// db.getCollection('conn').find({$and:[{id_orig_h:"XXX.XXX.XXX.XXX"},{id_resp_h:"XXX.XXX.XXX.XXX"}]}).count()
//
func (fs *FSImporter) bulkRemoveHugeUconns(datastore Datastore, targetDB string, filterHugeUconnsMap []uconnPair, connMap map[uconnPair]int) {
	var temp []*parsetypes.Temp
	resDB := fs.res.DB
	resConf := fs.res.Config
	var deleteQuery bson.M

	fmt.Println("\t[-] Removing unused connection info. This may take a while.")
	for _, uconn := range filterHugeUconnsMap {

		temp = append(temp, &parsetypes.Temp{
			Source:          uconn.src,
			Destination:     uconn.dst,
			ConnectionCount: connMap[uconn],
		})

		deleteQuery = bson.M{
			"$and": []bson.M{
				bson.M{"id_orig_h": uconn.src},
				bson.M{"id_resp_h": uconn.dst},
			}}
		bulkDeleteMany(deleteQuery, resDB, resConf, targetDB, resConf.T.Structure.ConnTable)
	}
	writerTemp(temp, resDB, resConf, targetDB)
}

// db.getCollection('conn').deleteMany({$and:[{id_orig_h:"XXX.XXX.XXX.XXX"},{id_resp_h:"XXX.XXX.XXX.XXX"}]})
func bulkDeleteMany(query bson.M, resDB *database.DB, resConf *config.Config, targetDB string, targetCL string) {
	ssn := resDB.Session.Copy()
	defer ssn.Close()

	info, err := ssn.DB(targetDB).C(targetCL).RemoveAll(query)

	if err != nil {
		fmt.Println("error! ", err, info)
	}
}

func writerTemp(output []*parsetypes.Temp, resDB *database.DB, resConf *config.Config, targetDB string) {

	// buffer length controls amount of ram used while exporting
	bufferLen := resConf.S.Bro.ImportBuffer

	// Create a buffer to hold a portion of the results
	buffer := make([]interface{}, 0, bufferLen)
	// while we can still iterate through the data add to the buffer
	for _, data := range output {

		// if the buffer is full, send to the remote database and clear buffer
		if len(buffer) == bufferLen {

			err := bulkWriteTemp(buffer, resDB, resConf, targetDB)

			if err != nil && err.Error() != "invalid BulkError instance: no errors" {
				fmt.Println(buffer)
				fmt.Println("write error 2", err)
			}

			buffer = buffer[:0]

		}

		buffer = append(buffer, data)
	}

	//send any data left in the buffer to the remote database
	err := bulkWriteTemp(buffer, resDB, resConf, targetDB)
	if err != nil && err.Error() != "invalid BulkError instance: no errors" {
		fmt.Println(buffer)
		fmt.Println("write error 2", err)
	}

	err = createTempIndexes(resDB, resConf, targetDB)

}

// Create indexes for the temporary database
func createTempIndexes(resDB *database.DB, resConf *config.Config, targetDB string) error {
	ssn := resDB.Session.Copy()
	defer ssn.Close()
	coll := ssn.DB(targetDB).C("temp")

	// Use the source destination pair as a composite index
	srcDstIndex := mgo.Index{
		Key:  []string{"src", "dst"},
		Name: "srcDstPair",
	}

	err := coll.EnsureIndex(srcDstIndex)

	if err != nil {
		fmt.Println(err)
	}

	connCountIndex := mgo.Index{
		Key:  []string{"connection_count"},
		Name: "connCount",
	}

	err = coll.EnsureIndex(connCountIndex)

	if err != nil {
		fmt.Println(err)
	}

	return err
}

func bulkWriteTemp(buffer []interface{}, resDB *database.DB, resConf *config.Config, targetDB string) error {
	ssn := resDB.Session.Copy()
	defer ssn.Close()

	// set up for bulk write to database
	bulk := ssn.DB(targetDB).C("temp").Bulk()
	// writes can be sent out of order
	bulk.Unordered()
	// inserts everything in the buffer into the bulk write object as a list
	// of single interfaces
	bulk.Insert(buffer...)

	// runs all queued operations
	_, err := bulk.Run()

	return err
}

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
