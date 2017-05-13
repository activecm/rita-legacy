package parser3

import (
	"io/ioutil"
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
		res *database.Resources
	}
)

//NewFSImporter creates a new file system importer
func NewFSImporter(resources *database.Resources) *FSImporter {
	return &FSImporter{
		res: resources,
	}
}

//Run starts importing a given path into a datastore
func (fs *FSImporter) Run(path string, datastore *Datastore) {
	// track the time spent parsing
	start := time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"start_time": start.Format("2006-01-02 15:04:05"),
		},
	).Info("Starting filesystem import")

	//find all of the bro log paths
	files := readDir(path, fs.res.Log)

	//hash the files and get their stats
	indexFiles(files, 1, &fs.res.System.BroConfig)

	hashTime := time.Now()
	fs.res.Log.WithFields(
		log.Fields{
			"hash_time": hashTime.Format("2006-01-02 15:04:05"),
		},
	).Info("Finished collecting log file details")
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

func indexFiles(files []string, indexingThreads int, broCfg *config.BroCfg) []*IndexedFile {
	n := len(files)
	output := make([]*IndexedFile, n)
	chunkSize := n / indexingThreads
	indexingWG := new(sync.WaitGroup)
	if chunkSize < 1 {
		chunkSize = 1
	}

	for i := 0; i < n; i += chunkSize {
		indexingWG.Add(1)
		go func(files []string, indexedFiles []*IndexedFile, broCfg *config.BroCfg, wg *sync.WaitGroup, start int, stride int, length int) {
			for j := start; j < start+stride; j++ {
				if j >= length {
					break
				}
				indexedFile, err := newIndexedFile(files[j], broCfg)
				if err != nil {
					indexedFiles[j] = indexedFile
				}
			}
			wg.Done()
		}(files, output, broCfg, indexingWG, i, chunkSize, n)
	}
	indexingWG.Wait()
	return output
}
