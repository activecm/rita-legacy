package parser3

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/parser3/fileparsetypes"
)

type (
	//FSImporter provides the ability to import bro files from the file system
	FSImporter struct {
		res             *database.Resources
		indexingThreads int
		parseThreads    int
	}
)

//NewFSImporter creates a new file system importer
func NewFSImporter(resources *database.Resources,
	indexingThreads int, parseThreads int) *FSImporter {
	return &FSImporter{
		res:             resources,
		indexingThreads: indexingThreads,
		parseThreads:    parseThreads,
	}
}

//Run starts importing a given path into a datastore
func (fs *FSImporter) Run(datastore *MongoDatastore) {
	// track the time spent parsing
	start := time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"start_time": start.Format("2006-01-02 15:04:05"),
		},
	).Info("Starting filesystem import")

	//find all of the bro log paths
	files := readDir(fs.res.System.BroConfig.LogPath, fs.res.Log)

	//hash the files and get their stats
	indexedFiles := indexFiles(files, fs.indexingThreads, fs.res.System, fs.res.Log)

	indexTime := time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"current_time": indexTime.Format("2006-01-02 15:04:05"),
		},
	).Info("Finished collecting log file details")

	indexedFiles = removeOldFilesFromIndex(indexedFiles, fs.res.MetaDB, fs.res.Log)

	createNewDatabases(indexedFiles, fs.res.MetaDB, fs.res.Log)

	parseFiles(indexedFiles, fs.parseThreads, datastore, fs.res.Log)

	updateFilesIndex(indexedFiles, fs.res.MetaDB, fs.res.Log)

	parseTime := time.Now()

	fs.res.Log.WithFields(
		log.Fields{
			"current_time": parseTime.Format("2006-01-02 15:04:05"),
			"total_time":   parseTime.Sub(start).String(),
		},
	).Info("Finished parsing log files")
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
		if file.IsDir() {
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
	cfg *config.SystemConfig, logger *log.Logger) []*fileparsetypes.IndexedFile {
	n := len(files)
	output := make([]*fileparsetypes.IndexedFile, n)
	indexingWG := new(sync.WaitGroup)

	for i := 0; i < indexingThreads; i++ {
		indexingWG.Add(1)

		go func(files []string, indexedFiles []*fileparsetypes.IndexedFile,
			sysConf *config.SystemConfig, logger *log.Logger,
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
//threads to use to parse the files, a MogoDB datastore object to store
//the bro data in, and a logger to report errors and parses the bro files
//line by line into the database.
func parseFiles(indexedFiles []*fileparsetypes.IndexedFile, parsingThreads int,
	datastore *MongoDatastore, logger *log.Logger) {
	n := len(indexedFiles)
	parsingWG := new(sync.WaitGroup)

	for i := 0; i < parsingThreads; i++ {
		parsingWG.Add(1)

		go func(indexedFiles []*fileparsetypes.IndexedFile, logger *log.Logger, wg *sync.WaitGroup, start int, jump int, length int) {
			for j := start; j < length; j += jump {
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
					data := parseLine(
						fileScanner.Text(),
						indexedFiles[j].GetHeader(),
						indexedFiles[j].GetFieldMap(),
						indexedFiles[j].GetBroDataFactory(),
						logger,
					)

					if data != nil {
						datastore.store(importedData{broData: data, file: indexedFiles[j]})
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
	datastore.flush()
}

func removeOldFilesFromIndex(indexedFiles []*fileparsetypes.IndexedFile,
	metaDatabase *database.MetaDBHandle, logger *log.Logger) []*fileparsetypes.IndexedFile {
	var toReturn []*fileparsetypes.IndexedFile
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

//createNewDatabases updates the metaDB with the new target databases
func createNewDatabases(indexedFiles []*fileparsetypes.IndexedFile, metaDatabase *database.MetaDBHandle,
	logger *log.Logger) {
	var seen = make(map[string]bool)
	for _, file := range indexedFiles {
		if file == nil {
			continue
		}
		targetDB := file.TargetDatabase
		_, ok := seen[targetDB]
		if !ok {
			seen[targetDB] = true
			result, err := metaDatabase.GetDBMetaInfo(targetDB)
			//database already exists
			if err == nil {
				//database has already been analyzed
				if result.Analyzed {
					logger.WithFields(log.Fields{
						"path":     file.Path,
						"database": targetDB,
					}).Error("cannot parse file into already analyzed database")
					panic("Attempted to parse file into already analyzed database")
				} //else parsing new file into unanalyzed database which exists
			} else { //database doesn't exist
				metaDatabase.AddNewDB(targetDB)
			}
		}
	}
}

//updateFilesIndex updates the files collection in the metaDB with the newly parsed files
func updateFilesIndex(indexedFiles []*fileparsetypes.IndexedFile, metaDatabase *database.MetaDBHandle,
	logger *log.Logger) {
	err := metaDatabase.AddParsedFiles(indexedFiles)
	if err != nil {
		logger.Error("Could not update the list of parsed files")
	}
}
