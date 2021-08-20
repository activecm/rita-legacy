package files

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/parser/parsetypes"
	pt "github.com/activecm/rita/parser/parsetypes"
)

//newIndexedFile takes in a file path and the current resource bundle and opens up the
//file path and parses out some metadata
func newIndexedFile(filePath string, targetDB string, targetCID int,
	logger *log.Logger, conf *config.Config) (*IndexedFile, error) {

	toReturn := new(IndexedFile)
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

	scanner, closeScanner, err := GetFileScanner(fileHandle)
	defer closeScanner() // handles closing the underlying fileHandle (and any associate subprocesses)
	if err != nil {
		return toReturn, err
	}

	header, err := scanTSVHeader(scanner)
	if err != nil {
		return toReturn, err
	}
	toReturn.SetHeader(header)

	var broDataFactory func() pt.BroData
	if header.ObjType != "" {
		// TSV log files have the type in a header
		broDataFactory = pt.NewBroDataFactory(header.ObjType)
	} else if scanner.Err() == nil && len(scanner.Bytes()) > 0 && // no error and there is text
		json.Valid(scanner.Bytes()) {
		toReturn.SetJSON()
		// check if "_path" is provided in the JSON data
		// https://github.com/corelight/json-streaming-logs
		t := struct {
			Path string `json:"_path"`
		}{}
		json.Unmarshal(scanner.Bytes(), &t)
		broDataFactory = pt.NewBroDataFactory(t.Path)

		// otherwise JSON log files only have the type in the filename
		if broDataFactory == nil {
			broDataFactory = pt.NewBroDataFactory(filepath.Base(toReturn.Path))
		}
	}
	if broDataFactory == nil {
		return toReturn, errors.New("could not map file header to parse type")
	}
	toReturn.SetBroDataFactory(broDataFactory)

	var fieldMap ZeekHeaderIndexMap
	// there is no need for the fieldMap with JSON
	if !toReturn.IsJSON() {
		fieldMap, err = mapZeekHeaderToParseType(header, broDataFactory, logger)
		if err != nil {
			return toReturn, err
		}
		toReturn.SetFieldMap(fieldMap)
	}

	//parse first line
	var line parsetypes.BroData
	if toReturn.IsJSON() {
		line = ParseJSONLine(scanner.Bytes(), broDataFactory, logger)
	} else {
		line = ParseTSVLine(scanner.Text(), header, fieldMap, broDataFactory, logger)
	}

	if line == nil {
		return toReturn, errors.New("could not parse first line of file")
	}

	toReturn.TargetCollection = line.TargetCollection(&conf.T.Structure)
	if toReturn.TargetCollection == "" {
		return toReturn, errors.New("could not find a target collection for file")
	}

	toReturn.TargetDatabase = targetDB
	toReturn.CID = targetCID

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

//IndexFiles takes in a list of Zeek files, a number of threads, the target database, and target chunk ID and parses
//some metadata out of the files
func IndexFiles(files []string, indexingThreads int, targetDB string, targetCID int,
	logger *log.Logger, conf *config.Config) []*IndexedFile {
	n := len(files)
	output := make([]*IndexedFile, n)
	indexingWG := new(sync.WaitGroup)

	for i := 0; i < indexingThreads; i++ {
		indexingWG.Add(1)

		go func(files []string, indexedFiles []*IndexedFile, targetDB string, targetCID int,
			logger *log.Logger, conf *config.Config, wg *sync.WaitGroup,
			start int, jump int, length int) {

			for j := start; j < length; j += jump {
				indexedFile, err := newIndexedFile(files[j], targetDB, targetCID, logger, conf)
				if err != nil {
					// log file is likely unsupported or empty
					logger.WithFields(log.Fields{
						"file":  files[j],
						"error": err.Error(),
					}).Debug("An error was encountered while indexing a file.")
					continue
				}
				indexedFiles[j] = indexedFile
			}
			wg.Done()
		}(files, output, targetDB, targetCID, logger, conf, indexingWG, i, indexingThreads, n)
	}

	indexingWG.Wait()

	// remove all nil values from the slice
	errCount := 0
	indexedFiles := make([]*IndexedFile, 0, len(output))
	for _, file := range output {
		if file != nil {
			indexedFiles = append(indexedFiles, file)
		} else {
			errCount++
		}
	}
	if errCount == len(output) {
		fmt.Println("\n\t[!] No compatible logs found or all log files provided were empty.")
		fmt.Println("\t[-] Exiting...")
		os.Exit(0)
	}
	return indexedFiles
}
