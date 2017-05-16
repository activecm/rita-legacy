package parser3

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/parser3/fileparsetypes"
	"github.com/ocmdev/rita/parser3/parsetypes"
)

//newIndexedFile takes in a file path and the bro config and opens up the
//file path and parses out some metadata
func newIndexedFile(filePath string, config *config.SystemConfig, logger *log.Logger) (*fileparsetypes.IndexedFile, error) {
	toReturn := new(fileparsetypes.IndexedFile)
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

	broDataFactory := parsetypes.NewBroDataFactory(header.ObjType)
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

	toReturn.TargetCollection = line.TargetCollection(&config.StructureConfig)
	if toReturn.TargetCollection == "" {
		fileHandle.Close()
		return toReturn, errors.New("Could not find a target collection for file")
	}

	timeVal := reflect.ValueOf(line).Elem().Field(fieldMap["ts"]).Int()
	toReturn.LogTime = time.Unix(timeVal, 0)

	toReturn.TargetDatabase = getTargetDatabase(filePath, toReturn.LogTime, &config.BroConfig)
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

//getTargetDatabase assigns a database to a log file based on the path, parse
//time and the bro config
func getTargetDatabase(path string, ttim time.Time, broConfig *config.BroCfg) string {
	toReturn := ""

	// check the directory map
	for key, val := range broConfig.DirectoryMap {
		if strings.Contains(path, key) {
			toReturn = broConfig.DBPrefix + val
			break
		}
	}
	//If a default database is specified put it in there
	if toReturn == "" && broConfig.DefaultDatabase != "" {
		toReturn = broConfig.DBPrefix + broConfig.DefaultDatabase
	}

	if toReturn != "" && broConfig.UseDates {
		toReturn += "-" + fmt.Sprintf("%d-%02d-%02d",
			ttim.Year(), ttim.Month(), ttim.Day())
	}
	return toReturn
}
