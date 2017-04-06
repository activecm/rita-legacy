package parser

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/bglebrun/rita/database"

	log "github.com/Sirupsen/logrus"
)

type (
	// Watcher provides an interface to keep up with a directory
	Watcher struct {
		path        string                 // path to be watched
		log         *log.Logger            // logger for this module
		res         *database.Resources    // configuration and resources
		meta        *database.MetaDBHandle // Handle to the metadata
		threadCount int                    // the number of reader threads
	}
)

// indexFiles takes in a string array and returns an array of
// heap allocated IndexedFile objects
func (w *Watcher) indexFiles(filesToIndex []string) []*database.IndexedFile {
	var toReturn []*database.IndexedFile

	for _, file := range filesToIndex {
		finfo, err := os.Stat(file)
		if err != nil {
			w.log.WithFields(log.Fields{
				"error": err.Error(),
				"file":  file,
			}).Error("aborting file parsing for this file")
			continue
		}
		fileHash, err := getFileHash(file)
		if err != nil {
			w.log.WithFields(log.Fields{
				"error": err.Error(),
				"file":  file,
			}).Error("aborting file parsing for this file")
			continue
		}
		db, err := w.getDBName(file)
		if err != nil {
			w.log.WithFields(log.Fields{
				"error": err.Error(),
				"file":  file,
			}).Error("aborting file parsing for this file")
			continue
		}
		indexedFile := new(database.IndexedFile)
		indexedFile.Path = file
		indexedFile.Hash = fileHash
		indexedFile.Length = finfo.Size()
		indexedFile.Mod = finfo.ModTime()
		indexedFile.Database = w.res.System.BroConfig.DBPrefix + db
		toReturn = append(toReturn, indexedFile)
	}
	return toReturn
}

// NewWatcher takes a top level directory to watch and returns a watcher
func NewWatcher(res *database.Resources, threadCount int) *Watcher {
	return &Watcher{
		path:        res.System.BroConfig.LogPath,
		log:         res.Log,
		res:         res,
		meta:        res.MetaDB,
		threadCount: threadCount,
	}
}

// Run simply runs the subcomponents in order building out the database
func (w *Watcher) Run(dw *DocWriter) {
	// track time spent parsing
	start := time.Now()
	w.log.WithFields(log.Fields{
		"start_time": start.Format("2006-01-02 15:04:05"),
	}).Info("Starting run")

	// read in the directory given and build the files
	files := w.readDir(w.path)          // stores the files to parse
	indexedFiles := w.indexFiles(files) //create indexedFile objects

	// add our files to the metadata table as unparsed
	// Also this filters out any files that were previously parsed
	newFiles := w.meta.InsertNewIndexedFiles(indexedFiles)

	// grab a ssn
	ssn := w.res.DB.Session.Copy()
	defer ssn.Close()

	// start the document writer write loop
	dw.Start()

	// Create a wait group for the read threads
	readWG := new(sync.WaitGroup)

	// build a channel of files that need parsing
	toParse := make(chan *database.IndexedFile)

	// kick off threadCount reader threads
	for i := 0; i < w.threadCount; i++ {
		readWG.Add(1)

		go func(readWG *sync.WaitGroup, toParse chan *database.IndexedFile, dw *DocWriter) {
			defer readWG.Done()

			// grab a file to parse
			for {
				file, ok := <-toParse
				if !ok {
					break
				}

				// keep track of individual times spent parsing
				fileStart := time.Now()
				w.log.WithFields(log.Fields{
					"file": file.Path,
				}).Debug("processing")

				// Actually launch the parser
				// TODO: watch for errors here
				err := parseFile(file, dw, w.res)

				// finish thracking time for this file
				if err == nil {
					// Set the parse time and update the database field if it changed
					w.meta.MarkFileImported(file)

					w.log.WithFields(log.Fields{
						"minutes": time.Since(fileStart).Minutes(),
						"file":    file.Path,
					}).Info("completed file")
				}
			}
		}(readWG, toParse, dw)
	}

	// put the indexed files into the channel
	for _, curr := range newFiles {
		toParse <- curr
	}
	close(toParse)

	//Wait for reads to finish
	readWG.Wait()

	// if we'ere debugging note when we think we're done vs when the writer finishes
	w.log.Debug("Parsing completed waiting on writer")

	// flush will cause a close on the writers channel
	dw.Flush()

	// finish tracking total parsing time
	w.log.WithFields(log.Fields{
		"elapsed_minutes": time.Since(start).Minutes(),
	}).Info("import completed")
}

// getDBName attempts to use the map from the yaml file to parse out a db name
func (w *Watcher) getDBName(file string) (string, error) {
	// check the directory map
	for key, val := range w.res.System.BroConfig.DirectoryMap {
		if strings.Contains(file, key) {
			return val, nil
		}
	}
	//If a default database is specified put it in there
	if w.res.System.BroConfig.DefaultDatabase != "" {
		return w.res.System.BroConfig.DefaultDatabase, nil
	}

	return "", errors.New("Did not find a match in directory map")
}

// readDir recursively reads the directory looking for log and .gz files
func (w *Watcher) readDir(cpath string) []string {
	var toReturn []string

	files, err := ioutil.ReadDir(cpath)
	if err != nil {
		w.log.WithFields(log.Fields{
			"error": err.Error(),
			"path":  cpath,
		}).Error("Error when reading directory")
	}

	for _, file := range files {
		if file.IsDir() {
			toReturn = append(toReturn, w.readDir(path.Join(cpath, file.Name()))...)
		}
		if strings.HasSuffix(file.Name(), "gz") ||
			strings.HasSuffix(file.Name(), "log") {
			toReturn = append(toReturn, path.Join(cpath, file.Name()))
		}
	}
	return toReturn
}

// getFileHash computes an md5 hash of the file at filepath
func getFileHash(filepath string) (string, error) {

	var result string
	file, err := os.Open(filepath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	hash := md5.New()
	fstat, err := file.Stat()
	if err != nil {
		return result, err
	}

	if fstat.Size() >= 15000 {
		if _, err := io.CopyN(hash, file, 15000); err != nil {
			return result, err
		}
	} else {
		if _, err := io.Copy(hash, file); err != nil {
			return result, err
		}
	}
	var byteset []byte
	ret := fmt.Sprintf("%x", hash.Sum(byteset))
	return ret, nil
}
