package parser3

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ocmdev/rita/config"
)

//IndexedFile ties a file to a target collection and database
type IndexedFile struct {
	Path             string
	Length           int64
	ModTime          time.Time
	Hash             string
	TargetCollection string
	TargetDatabase   string
	LogTime          time.Time
	ParseTime        time.Time
}

func newIndexedFile(filePath string, broCfg *config.BroCfg) (*IndexedFile, error) {
	toReturn := new(IndexedFile)
	toReturn.Path = filePath

	fileHandle, err := os.Open(filePath)
	if err != nil {
		return toReturn, err
	}

	fInfo, err := fileHandle.Stat()
	if err != nil {
		return toReturn, err
	}
	toReturn.Length = fInfo.Size()
	toReturn.ModTime = fInfo.ModTime()

	fHash, err := getFileHash(fileHandle, fInfo)
	if err != nil {
		return toReturn, err
	}
	toReturn.Hash = fHash

	tgtColl, tgtDB, err := getTargetCollectionAndDB(fileHandle, broCfg)
	if err != nil {
		return toReturn, err
	}
	toReturn.TargetCollection = tgtColl
	toReturn.TargetDatabase = tgtDB
	fileHandle.Close()
	return toReturn, nil
}

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

func getTargetCollectionAndDB(fileHandle *os.File, broCfg *config.BroCfg) (string, string, error) {
	//TODO: Write parsing code to get the target collection and database
	return "", "", nil
}
