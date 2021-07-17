package files

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	pt "github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/util"

	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

// GatherLogFiles reads the files and directories looking for log and gz files
func GatherLogFiles(paths []string, logger *log.Logger) []string {
	var toReturn []string

	for _, path := range paths {
		if util.IsDir(path) {
			toReturn = append(toReturn, gatherDir(path, logger)...)
		} else if strings.HasSuffix(path, ".gz") ||
			strings.HasSuffix(path, ".log") {
			toReturn = append(toReturn, path)
		} else {
			logger.WithFields(log.Fields{
				"path": path,
			}).Warn("Ignoring non .log or .gz file")
		}
	}

	return toReturn
}

// gatherDir reads the directory looking for log and .gz files
func gatherDir(cpath string, logger *log.Logger) []string {
	var toReturn []string
	files, err := ioutil.ReadDir(cpath)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
			"path":  cpath,
		}).Error("Error when reading directory")
	}

	for _, file := range files {
		// Stop RITA from following symlinks
		// In the case that RITA is pointed directly at Bro, it should not
		// parse the "current" symlink which points to the spool.
		// if file.IsDir() && file.Mode() != os.ModeSymlink {
		// 	toReturn = append(toReturn, readDir(path.Join(cpath, file.Name()), logger)...)
		// }
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".gz") ||
			strings.HasSuffix(file.Name(), ".log") {
			toReturn = append(toReturn, path.Join(cpath, file.Name()))
		}
	}
	return toReturn
}

// GetFileScanner returns a buffered file scanner for a bro log file, a function to close the
// underlying stream and any associated processors, as well as any error that may occur while
// creating the scanner
func GetFileScanner(fileHandle *os.File) (scanner *bufio.Scanner, closer func() error, err error) {
	// by default just close out the underlying file handle
	closer = fileHandle.Close

	ftype := fileHandle.Name()[len(fileHandle.Name())-3:]
	if ftype != ".gz" && ftype != "log" {
		return nil, closer, errors.New("filetype not recognized")
	}

	if ftype == ".gz" {
		var gzipReader io.Reader
		gzipReader, closer, err = newGzipReader(fileHandle)
		if err != nil {
			return nil, closer, err
		}
		scanner = bufio.NewScanner(gzipReader)
	} else {
		scanner = bufio.NewScanner(fileHandle)
	}

	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return scanner, closer, nil
}

//newGzipReader returns an un-gzipped byte stream given a gzip compressed byte stream.
//This method tries to use the system's pigz or gzip implementation before relying on
//Golang's gzip package (as it is quite slow). Returns stream to read from, a function to
//close the underlying stream, and any err that may occur when opening the stream.
func newGzipReader(fileHandle io.ReadCloser) (reader io.Reader, closer func() error, err error) {
	// by default just close out the underlying file handle
	// works for built in gzip library and error cases
	closer = fileHandle.Close

	var gzipPath string
	if path, err := exec.LookPath("pigz"); err == nil {
		gzipPath = path
	} else if path, err := exec.LookPath("gzip"); err == nil {
		gzipPath = path
	} else {
		// can't find system command, use golang lib, no special closing logic needed other than
		// to close the underlying file descriptor
		reader, err = gzip.NewReader(fileHandle)
		return reader, closer, err
	}

	// create the subprocess
	ctx, cancel := context.WithCancel(context.Background())
	gzipCommand := exec.CommandContext(ctx, gzipPath, "-d", "-c")

	// tell the subprocess to read from the given stream
	gzipCommand.Stdin = fileHandle

	// return/ pipe the output back out to the caller
	pipeR, err := gzipCommand.StdoutPipe()
	if err != nil {
		cancel() // essentially a no-op.  makes the linter happy tho.
		return reader, fileHandle.Close, err
	}

	var cmdStdErr bytes.Buffer
	gzipCommand.Stderr = &cmdStdErr

	if err := gzipCommand.Start(); err != nil {
		cancel() // essentially a no-op.  makes the linter happy tho.
		return reader, fileHandle.Close, err
	}

	// update the closer to kill the subprocess in addition to closing the file descriptor
	closer = func() error {
		// kill the subprocess, any errors will come out on the read side or during Wait
		cancel()
		// close the file that was passed in
		errFile := fileHandle.Close()
		// wait for the subprocess to finish out
		errProc := gzipCommand.Wait()

		// add StdErr to the process error if the command returned a nonzero code
		if errProc != nil && cmdStdErr.Len() > 0 {
			errProc = fmt.Errorf("%s: %s", errProc.Error(), cmdStdErr.String())
		}

		// handle return errors up
		if errProc != nil && errFile != nil {
			return fmt.Errorf("%s; %s", errProc.Error(), errFile.Error())
		}
		if errProc != nil {
			return errProc
		}
		if errFile != nil {
			return errFile
		}
		return nil
	}

	return pipeR, closer, nil
}

// scanHeader scans the comment lines out of a bro file and returns a
// BroHeader object containing the information. NOTE: This has the side
// effect of advancing the fileScanner so that fileScanner.Text() will
// return the first log entry in the file.
func scanTSVHeader(fileScanner *bufio.Scanner) (*BroHeader, error) {
	toReturn := new(BroHeader)
	for fileScanner.Scan() {
		if fileScanner.Err() != nil {
			break
		}
		if len(fileScanner.Bytes()) < 1 {
			continue
		}
		//On the comment lines
		if fileScanner.Bytes()[0] == '#' {
			line := strings.Fields(fileScanner.Text())
			switch line[0][1:] {
			case "separator":
				var err error
				toReturn.Separator, err = strconv.Unquote("\"" + line[1] + "\"")
				if err != nil {
					return toReturn, err
				}
			case "set_separator":
				toReturn.SetSep = line[1]
			case "empty_field":
				toReturn.Empty = line[1]
			case "unset_field":
				toReturn.Unset = line[1]
			case "fields":
				toReturn.Names = line[1:]
			case "types":
				toReturn.Types = line[1:]
			case "path":
				toReturn.ObjType = line[1]
			}
		} else {
			//We are done parsing the comments
			break
		}
	}

	if len(toReturn.Names) != len(toReturn.Types) {
		return toReturn, errors.New("name / type mismatch")
	}
	return toReturn, nil
}

func mapZeekHeaderToParseType(header *BroHeader, broDataFactory func() pt.BroData, logger *log.Logger) (ZeekHeaderIndexMap, error) {
	broData := broDataFactory()
	structType := reflect.TypeOf(broData).Elem()

	indexMap := ZeekHeaderIndexMap{
		NthLogFieldExistsInParseType: make([]bool, len(header.Names)),
		NthLogFieldParseTypeOffset:   make([]int, len(header.Names)),
	}

	// parseTypeFieldInfo and the parseTypeFields map record the names, types, and offsets of the
	// Zeek fields we want to populate the broData with. Recording this info in a map allows
	// us to match the Zeek header to the parse type fields without nested loops.
	type parseTypeFieldInfo struct {
		zeekType             string
		parseTypeFieldOffset int
	}
	// parseTypeFields maps from Zeek field names to the associated info as defined by the
	// broData struct tags
	parseTypeFields := make(map[string]parseTypeFieldInfo)

	// walk the fields of the broData, making sure the broData struct has
	// an equal number of named bro fields and bro types
	for i := 0; i < structType.NumField(); i++ {
		structField := structType.Field(i)
		zeekName := structField.Tag.Get("bro")
		zeekType := structField.Tag.Get("brotype")

		//If this field is not associated with bro, skip it
		if len(zeekName) == 0 && len(zeekType) == 0 {
			continue
		}

		if len(zeekName) == 0 || len(zeekType) == 0 {
			return indexMap, errors.New("incomplete bro variable")
		}

		parseTypeFields[zeekName] = parseTypeFieldInfo{
			zeekType:             zeekType,
			parseTypeFieldOffset: i,
		}
	}

	for index, name := range header.Names {
		fieldInfo, ok := parseTypeFields[name]
		if !ok {
			//an unmatched field which exists in the log but not the struct
			//is not a fatal error, so we report it and move on
			logger.WithFields(log.Fields{
				"error":         "unmatched field in log",
				"missing_field": name,
			}).Info("the log contains a field with no candidate in the data structure")
			continue
		}

		if header.Types[index] != fieldInfo.zeekType {
			err := errors.New("type mismatch found in log")
			logger.WithFields(log.Fields{
				"error":         err,
				"type in log":   header.Types[index],
				"expected type": fieldInfo.zeekType,
			})
			return indexMap, err
		}

		indexMap.NthLogFieldExistsInParseType[index] = true
		indexMap.NthLogFieldParseTypeOffset[index] = fieldInfo.parseTypeFieldOffset
	}

	return indexMap, nil
}

//ParseJSONLine creates a new BroData from a line of a Zeek JSON log.
func ParseJSONLine(lineBuffer []byte, broDataFactory func() pt.BroData,
	logger *log.Logger) pt.BroData {

	dat := broDataFactory()
	err := jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(lineBuffer, dat)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Encountered unparsable JSON in log")
	}
	dat.ConvertFromJSON()
	return dat
}

func parseTSVField(fieldText string, fieldType string, targetField reflect.Value, logger *log.Logger) {
	switch fieldType {
	case pt.Time:
		decimalPointIdx := strings.Index(fieldText, ".")
		if decimalPointIdx == -1 {
			logger.WithFields(log.Fields{
				"error": "no decimal point found in timestamp",
				"value": fieldText,
			}).Error("Couldn't convert unix ts")
			targetField.SetInt(-1)
			return
		}

		s, err := strconv.Atoi(fieldText[:decimalPointIdx])
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err.Error(),
				"value": fieldText,
			}).Error("Couldn't convert unix ts")
			targetField.SetInt(-1)
			return
		}

		nanos, err := strconv.Atoi(fieldText[decimalPointIdx+1:])
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err.Error(),
				"value": fieldText,
			}).Error("Couldn't convert unix ts")
			targetField.SetInt(-1)
			return
		}

		ttim := time.Unix(int64(s), int64(nanos))
		tval := ttim.Unix()
		targetField.SetInt(tval)
	case pt.String:
		fallthrough
	case pt.Enum:
		fallthrough
	case pt.Addr:
		targetField.SetString(fieldText)
	case pt.Port:
		fallthrough
	case pt.Count:
		intValue, err := strconv.Atoi(fieldText)
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err.Error(),
				"value": fieldText,
			}).Error("Couldn't convert port number/ count")
			targetField.SetInt(-1)
			return
		}
		targetField.SetInt(int64(intValue))
	case pt.Interval:
		flt, err := strconv.ParseFloat(fieldText, 64)
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err.Error(),
				"value": fieldText,
			}).Error("Couldn't convert float")
			targetField.SetFloat(-1.0)
			return
		}
		targetField.SetFloat(flt)
	case pt.Bool:
		if fieldText == "T" {
			targetField.SetBool(true)
		} else {
			targetField.SetBool(false)
		}
	case pt.StringSet:
		fallthrough
	case pt.EnumSet:
		fallthrough
	case pt.StringVector:
		tokens := strings.Split(fieldText, ",")
		tVal := reflect.ValueOf(tokens)
		targetField.Set(tVal)
	case pt.IntervalVector:
		tokens := strings.Split(fieldText, ",")
		floats := make([]float64, len(tokens))
		for i, val := range tokens {
			var err error
			floats[i], err = strconv.ParseFloat(val, 64)
			if err != nil {
				logger.WithFields(log.Fields{
					"error": err.Error(),
					"value": val,
				}).Error("Couldn't convert float")
				return
			}
		}
		fVal := reflect.ValueOf(floats)
		targetField.Set(fVal)
	default:
		logger.WithFields(log.Fields{
			"error": "Unhandled type",
			"value": fieldType,
		}).Error("Encountered unhandled type in log")
	}
}

//ParseTSVLine creates a new BroData from a line of a Zeek TSV log.
//String matching is generally faster than byte matching in Golang for some reason, so we take use a string
//rather than bytes here.
func ParseTSVLine(lineString string, header *BroHeader,
	fieldMap ZeekHeaderIndexMap, broDataFactory func() pt.BroData,
	logger *log.Logger) pt.BroData {

	if strings.HasPrefix(lineString, "#") {
		return nil
	}

	dat := broDataFactory()
	data := reflect.ValueOf(dat).Elem()

	tokenEndIdx := strings.Index(lineString, header.Separator)
	tokenCounter := 0
	for tokenEndIdx != -1 && tokenCounter < len(header.Names) {
		//fields not in the struct will not be parsed
		if lineString[:tokenEndIdx] != header.Empty && lineString[:tokenEndIdx] != header.Unset {
			// we used to map from the field names to their field offsets in the broData, but
			// since this code is very hot, it was replaced with the array accesses within the
			// fieldMap struct seen below. Now, we map from the field's index in the file header
			// to the offsets in the broData using the NthLogFieldParseTypeOffset array.
			if fieldMap.NthLogFieldExistsInParseType[tokenCounter] {
				parseTSVField(
					lineString[:tokenEndIdx],
					header.Types[tokenCounter],
					data.Field(fieldMap.NthLogFieldParseTypeOffset[tokenCounter]),
					logger,
				)
			}
		}

		// chomp off the portion we just parsed
		lineString = lineString[tokenEndIdx+len(header.Separator):]
		tokenEndIdx = strings.Index(lineString, header.Separator)
		tokenCounter++
	}

	//handle last field
	if lineString != header.Empty && lineString != header.Unset {
		if fieldMap.NthLogFieldExistsInParseType[tokenCounter] {
			parseTSVField(
				lineString,
				header.Types[tokenCounter],
				data.Field(fieldMap.NthLogFieldParseTypeOffset[tokenCounter]),
				logger,
			)
		}
	}

	return dat
}
