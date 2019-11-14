package parser

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/activecm/rita/resources"
	"io"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/activecm/rita/database"
	fpt "github.com/activecm/rita/parser/fileparsetypes"
	pt "github.com/activecm/rita/parser/parsetypes"
)

//newIndexedFile takes in a file path and the current resource bundle and opens up the
//file path and parses out some metadata
func newIndexedFile(filePath string, res *resources.Resources) (*fpt.IndexedFile, error) {
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

	fieldMap, err := mapBroHeaderToParserType(header, broDataFactory, res.Log)
	if err != nil {
		fileHandle.Close()
		return toReturn, err
	}
	toReturn.SetFieldMap(fieldMap)

	//parse first line
	line := parseLine(scanner.Text(), header, fieldMap, broDataFactory, res.Log)
	if line == nil {
		fileHandle.Close()
		return toReturn, errors.New("Could not parse first line of file for time")
	}

	toReturn.TargetCollection = line.TargetCollection(&res.Config.T.Structure)
	if toReturn.TargetCollection == "" {
		fileHandle.Close()
		return toReturn, errors.New("Could not find a target collection for file")
	}

	toReturn.TargetDatabase = res.DB.GetSelectedDB()
	toReturn.CID = res.Config.S.Rolling.CurrentChunk

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

//indexFiles takes in a list of bro files, a number of threads, and parses
//some metadata out of the files
func indexFiles(files []string, indexingThreads int, res *resources.Resources) []*fpt.IndexedFile {
	n := len(files)
	output := make([]*fpt.IndexedFile, n)
	indexingWG := new(sync.WaitGroup)

	for i := 0; i < indexingThreads; i++ {
		indexingWG.Add(1)

		go func(files []string, indexedFiles []*fpt.IndexedFile,
			res *resources.Resources, wg *sync.WaitGroup,
			start int, jump int, length int) {

			for j := start; j < length; j += jump {
				indexedFile, err := newIndexedFile(files[j], res)
				if err != nil {
					res.Log.WithFields(log.Fields{
						"file":  files[j],
						"error": err.Error(),
					}).Debug("An error was encountered while indexing a file")
					//errored on files will be nil
					continue
				}
				indexedFiles[j] = indexedFile
			}
			wg.Done()
		}(files, output, res, indexingWG, i, indexingThreads, n)
	}

	indexingWG.Wait()

	// remove all nil values from the slice
	indexedFiles := make([]*fpt.IndexedFile, 0, len(output))
	for _, file := range output {
		if file != nil {
			indexedFiles = append(indexedFiles, file)
		}
	}

	return indexedFiles
}

//removeOldFilesFromIndex checks all indexedFiles passed in to ensure
//that they have not previously been imported into the same database.
//The files are compared based on their hashes (md5 of first 15000 bytes)
//and the database they are slated to be imported into.
func removeOldFilesFromIndex(indexedFiles []*fpt.IndexedFile,
	metaDatabase *database.MetaDB, logger *log.Logger, targetDatabase string) []*fpt.IndexedFile {
	var toReturn []*fpt.IndexedFile
	oldFiles, err := metaDatabase.GetFiles(targetDatabase)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not obtain a list of previously parsed files")
	}

	for _, newFile := range indexedFiles {
		have := false
		for _, oldFile := range oldFiles {
			if oldFile.Hash == newFile.Hash {
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
