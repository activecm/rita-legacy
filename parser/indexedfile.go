package parser

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	fpt "github.com/activecm/rita/parser/fileparsetypes"
	pt "github.com/activecm/rita/parser/parsetypes"
)

//newIndexedFile takes in a file path and the bro config and opens up the
//file path and parses out some metadata
func newIndexedFile(filePath string, config *config.Config,
	logger *log.Logger) (*fpt.IndexedFile, error) {
	toReturn := new(fpt.IndexedFile)
	toReturn.Path = filePath

	fileHandle, err := os.Open(filePath)
	if err != nil {
		return toReturn, err
	}

	fInfo, err := fileHandle.Stat()
	if err != nil {
		fileHandle.Close()
		return toReturn, err
	}
	toReturn.Length = fInfo.Size()
	toReturn.ModTime = fInfo.ModTime()

	fHash, err := getFileHash(fileHandle, fInfo)
	if err != nil {
		fileHandle.Close()
		return toReturn, err
	}
	toReturn.Hash = fHash

	scanner, err := getFileScanner(fileHandle)
	if err != nil {
		fileHandle.Close()
		return toReturn, err
	}

	header, err := scanHeader(scanner)
	if err != nil {
		fileHandle.Close()
		return toReturn, err
	}
	toReturn.SetHeader(header)

	broDataFactory := pt.NewBroDataFactory(header.ObjType)
	if broDataFactory == nil {
		fileHandle.Close()
		return toReturn, errors.New("Could not map file header to parse type")
	}
	toReturn.SetBroDataFactory(broDataFactory)

	fieldMap, err := mapBroHeaderToParserType(header, broDataFactory, logger)
	if err != nil {
		fileHandle.Close()
		return toReturn, err
	}
	toReturn.SetFieldMap(fieldMap)

	//parse first line
	line := parseLine(scanner.Text(), header, fieldMap, broDataFactory, logger)
	if line == nil {
		fileHandle.Close()
		return toReturn, errors.New("Could not parse first line of file for time")
	}

	toReturn.TargetCollection = line.TargetCollection(&config.T.Structure)
	if toReturn.TargetCollection == "" {
		fileHandle.Close()
		return toReturn, errors.New("Could not find a target collection for file")
	}

	toReturn.TargetDatabase = getTargetDatabase(filePath, &config.S.Bro)
	if toReturn.TargetDatabase == "" {
		fileHandle.Close()
		return toReturn, errors.New("Could not find a dataset for file")
	}

	fileHandle.Close()
	return toReturn, nil
}

//getFileHash md5's the first 15000 bytes of a file
func getFileHash(fileHandle *os.File, fInfo os.FileInfo) (string, error) {
	hash := md5.New()

	if fInfo.Size() >= 15000 {
		if _, err := io.CopyN(hash, fileHandle, 15000); err != nil {
			return "", err
		}
	} else {
		if _, err := io.Copy(hash, fileHandle); err != nil {
			return "", err
		}
	}
	//be nice and reset the file handle
	fileHandle.Seek(0, 0)
	var byteset []byte
	return fmt.Sprintf("%x", hash.Sum(byteset)), nil
}

//getTargetDatabase assigns a database to a log file based on the path,
//and the bro config
func getTargetDatabase(filePath string, broConfig *config.BroStaticCfg) string {
	var targetDatabase bytes.Buffer
	targetDatabase.WriteString(broConfig.DBRoot)
	//Append subfolders to target db
	relativeStartIndex := len(broConfig.ImportDirectory)
	pathSep := string(os.PathSeparator)
	relativePath := filePath[relativeStartIndex+len(pathSep):]

	//This routine uses Split rather than substring (0, index of path sep)
	//because we may wish to add all the subdirectories to the db prefix
	pathPieces := strings.Split(relativePath, pathSep)
	//if there is more than just the file name
	if len(pathPieces) > 1 {
		targetDatabase.WriteString("-")
		targetDatabase.WriteString(pathPieces[0])
	}
	return targetDatabase.String()
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
