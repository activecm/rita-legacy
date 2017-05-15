package parser3

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
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

	hashTime := time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"hash_time": hashTime.Format("2006-01-02 15:04:05"),
		},
	).Info("Finished collecting log file details")

	parseFiles(indexedFiles, fs.parseThreads, datastore, fs.res.Log)

	fs.res.Log.WithFields(
		log.Fields{
			"parse_time": hashTime.Format("2006-01-02 15:04:05"),
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
	cfg *config.SystemConfig, logger *log.Logger) []*IndexedFile {
	n := len(files)
	output := make([]*IndexedFile, n)
	indexingWG := new(sync.WaitGroup)

	for i := 0; i < indexingThreads; i++ {
		indexingWG.Add(1)

		go func(files []string, indexedFiles []*IndexedFile,
			sysConf *config.SystemConfig, logger *log.Logger,
			wg *sync.WaitGroup, start int, jump int, length int) {

			for j := start; j < length; j += jump {
				indexedFile, err := newIndexedFile(files[j], cfg, logger)
				if err != nil {
					logger.WithFields(log.Fields{
						"file":  files[j],
						"error": err.Error(),
					}).Error("An error was encountered while indexing a file")
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

func parseFiles(indexedFiles []*IndexedFile, parsingThreads int,
	datastore *MongoDatastore, logger *log.Logger) {
	n := len(indexedFiles)
	parsingWG := new(sync.WaitGroup)

	for i := 0; i < parsingThreads; i++ {
		parsingWG.Add(1)

		go func(indexedFiles []*IndexedFile, logger *log.Logger, wg *sync.WaitGroup, start int, jump int, length int) {
			for j := start; j < length; j += jump {
				if indexedFiles[j] == nil {
					continue
				}
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
						indexedFiles[j].header,
						indexedFiles[j].fieldMap,
						indexedFiles[j].broDataFactory,
						logger,
					)

					if data != nil {
						datastore.store(importedData{broData: data, file: indexedFiles[j]})
					} else {
						fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
						fmt.Println(fileScanner.Text())
					}
				}
				indexedFiles[j].ParseTime = time.Now()
				fileHandle.Close()
			}
			wg.Done()
		}(indexedFiles, logger, parsingWG, i, parsingThreads, n)

	}
	parsingWG.Wait()
	datastore.flush()
}
