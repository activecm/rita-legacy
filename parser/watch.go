package parser

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/parser/docwriter"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

type (
	// Watcher provides an interface to keep up with a directory
	Watcher struct {
		path         string                 // path to be watched
		filesToParse []string               // files that have not been parsed
		log          *log.Logger            // logger for this module
		files        []*database.PFile      // files with full stat info
		cfg          *config.Resources      // configuration and resources
		Meta         *database.MetaDBHandle // Handle to the metadata

	}
)

// newPFiles generates the pfileObjects
func (w *Watcher) newPFiles() {
	for _, file := range w.filesToParse {
		finfo, err := os.Stat(file)
		if err != nil {
			w.log.WithFields(log.Fields{
				"error": err.Error(),
				"file":  file,
			}).Error("Couldn't stat file")
			continue
		}
		fhash, err := getFileHash(file)
		if err != nil {
			w.log.WithFields(log.Fields{
				"error": err.Error(),
				"file":  file,
			}).Error("failed to hash file")
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
		pfile := &database.PFile{
			Path:     file,
			Hash:     fhash,
			Length:   finfo.Size(),
			Mod:      finfo.ModTime(),
			DataBase: db,
		}
		w.files = append(w.files, pfile)
	}
}

// NewWatcher takes a top level directory to watch and returns a watcher
func NewWatcher(cfg *config.Resources, dbm *database.MetaDBHandle) *Watcher {
	return &Watcher{
		path: cfg.System.BroConfig.LogPath,
		log:  cfg.Log,
		cfg:  cfg,
		Meta: dbm,
	}
}

// Run simply runs the subcomponents in order building out the database
func (w *Watcher) Run(dw *docwriter.DocWriter) {
	// track time spent parsing
	start := time.Now()
	w.log.WithFields(log.Fields{
		"start_time": start.Format("2006-01-02 15:04:05"),
	}).Info("Starting run")

	// read in the directory given and build the files
	w.readDir(w.path)
	w.newPFiles()

	// add our files to the metadata table as unparsed
	_ = w.Meta.UpdateFiles(w.files)

	// grab a ssn key
	ssn := w.cfg.Session.Copy()
	defer ssn.Close()

	// build a channel of files that need parsing
	toParse := make(chan *database.PFile, len(w.files))

	// check to see if this file is already parsed, if not add to channel
	for _, curr := range w.files {
		if curr.Parsed > 0 {
			continue
		}
		toParse <- curr
	}

	// close the channel (avoid deadlocking)
	close(toParse)

	// waitgroup for parsers
	wg := new(sync.WaitGroup)

	for {
		// grab a file to parse
		file, ok := <-toParse
		if !ok {
			break
		}

		// if we found a file add one to the syncro and drop into our parsing logic
		wg.Add(1)
		go func(wg *sync.WaitGroup,
			f *database.PFile,
			dw *docwriter.DocWriter) {

			// makesure waitgroup.Done() gets called when we exit this function
			defer wg.Done()

			// keep track of individual times spent parsing
			myStart := time.Now()
			w.log.WithFields(log.Fields{
				"file":       f.Path,
				"start_time": myStart.Format("2006-01-02 15:04:05"),
			}).Info("processing")

			// get a database name possibly based on the name of the path and date
			dben, err := w.getDBName(f.Path)
			if err != nil {
				w.log.WithFields(log.Fields{
					"error": err.Error(),
					"file":  f.Path,
				}).Error("aborting")
				return
			}

			// Actually launch the parser
			ParseFile(f.Path, dw, w.cfg, dben)

			// TODO track this error and do something meaningful with it
			_ = w.Meta.MarkCompleted(f)

			// finish thracking time for this file
			w.log.WithFields(log.Fields{
				"minutes": time.Since(myStart).Minutes(),
				"file":    f.Path,
			}).Info("completed file")

		}(wg, file, dw)
	}

	// wait for all of the files to finish parsing
	wg.Wait()

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
	var res string
	for key, val := range w.cfg.System.BroConfig.DirectoryMap {
		if strings.Contains(file, key) {
			return val, nil
		}
	}
	return res, errors.New("Did not find a match in directory map")
}

// readDir recursively reads the directory looking for log and .gz files
func (w *Watcher) readDir(cpath string) {
	files, err := ioutil.ReadDir(cpath)
	if err != nil {
		w.log.WithFields(log.Fields{
			"error": err.Error(),
			"path":  cpath,
		}).Error("Error when reading directory")
	}
	for _, file := range files {
		if file.IsDir() {
			w.readDir(path.Join(cpath, file.Name()))
		}
		if strings.HasSuffix(file.Name(), "gz") ||
			strings.HasSuffix(file.Name(), "log") {
			w.filesToParse = append(w.filesToParse, path.Join(cpath, file.Name()))
		}
	}
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
