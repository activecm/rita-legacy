package files

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	pt "github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/util"
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

// GetFileScanner returns a buffered file scanner for a bro log file
func GetFileScanner(fileHandle *os.File) (*bufio.Scanner, error) {
	ftype := fileHandle.Name()[len(fileHandle.Name())-3:]
	if ftype != ".gz" && ftype != "log" {
		return nil, errors.New("filetype not recognized")
	}

	var scanner *bufio.Scanner
	if ftype == ".gz" {
		rdr, err := gzip.NewReader(fileHandle)
		if err != nil {
			return nil, err
		}
		scanner = bufio.NewScanner(rdr)
	} else {
		scanner = bufio.NewScanner(fileHandle)
	}

	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return scanner, nil
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

//mapBroHeaderToParserType checks a parsed BroHeader against
//a BroData struct and returns a mapping from bro field names in the
//bro header to the indexes of the respective fields in the BroData struct
func mapBroHeaderToParserType(header *BroHeader, broDataFactory func() pt.BroData,
	logger *log.Logger) (BroHeaderIndexMap, error) {
	// The lookup struct gives us a way to walk the data structure only once
	type lookup struct {
		broType string
		offset  int
	}

	//create a bro data to check the header against
	broData := broDataFactory()

	// map the bro names -> the brotypes
	fieldTypes := make(map[string]lookup)

	//toReturn is a simplified version of the fieldTypes map which
	//links a bro field name to its index in the broData struct
	toReturn := make(map[string]int)

	structType := reflect.TypeOf(broData).Elem()

	// walk the fields of the bro data, making sure the bro data struct has
	// an equal number of named bro fields and bro type
	for i := 0; i < structType.NumField(); i++ {
		structField := structType.Field(i)
		broName := structField.Tag.Get("bro")
		broType := structField.Tag.Get("brotype")

		//If this field is not associated with bro, skip it
		if len(broName) == 0 && len(broType) == 0 {
			continue
		}

		if len(broName) == 0 || len(broType) == 0 {
			return nil, errors.New("incomplete bro variable")
		}
		fieldTypes[broName] = lookup{broType: broType, offset: i}
		toReturn[broName] = i
	}

	// walk the header names array and link each field up with a type in the
	// bro data
	for index, name := range header.Names {
		lu, ok := fieldTypes[name]
		if !ok {
			//NOTE: an unmatched field which exists in the log but not the struct
			//is not a fatal error, so we report it and move on
			logger.WithFields(log.Fields{
				"error":         "unmatched field in log",
				"missing_field": name,
			}).Info("the log contains a field with no candidate in the data structure")
			continue
		}

		if header.Types[index] != lu.broType {
			err := errors.New("type mismatch found in log")
			logger.WithFields(log.Fields{
				"error":               err,
				"header.Types[index]": header.Types[index],
				"lu.broType":          lu.broType,
			})
			return nil, err
		}
	}

	return toReturn, nil
}

//ParseJSONLine creates a new BroData from a line of a Zeek JSON log.
func ParseJSONLine(lineBuffer []byte, broDataFactory func() pt.BroData,
	logger *log.Logger) pt.BroData {

	dat := broDataFactory()
	err := json.Unmarshal(lineBuffer, dat)
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
		secs := strings.Split(fieldText, ".")
		s, err := strconv.ParseInt(secs[0], 10, 64)
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err.Error(),
				"value": fieldText,
			}).Error("Couldn't convert unix ts")
			targetField.SetInt(-1)
			return
		}

		nanos, err := strconv.ParseInt(secs[1], 10, 64)
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err.Error(),
				"value": fieldText,
			}).Error("Couldn't convert unix ts")
			targetField.SetInt(-1)
			return
		}

		ttim := time.Unix(s, nanos)
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
		intValue, err := strconv.ParseInt(fieldText, 10, 32)
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err.Error(),
				"value": fieldText,
			}).Error("Couldn't convert port number/ count")
			targetField.SetInt(-1)
			return
		}
		targetField.SetInt(intValue)
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
func ParseTSVLine(lineString string, header *BroHeader,
	fieldMap BroHeaderIndexMap, broDataFactory func() pt.BroData,
	logger *log.Logger) pt.BroData {

	if strings.HasPrefix(lineString, "#") {
		return nil
	}

	dat := broDataFactory()
	data := reflect.ValueOf(dat).Elem()

	tokenEndIdx := strings.Index(lineString, header.Separator)
	tokenCounter := 0
	for tokenEndIdx != -1 {
		//fields not in the struct will not be parsed
		if lineString[:tokenEndIdx] != header.Empty && lineString[:tokenEndIdx] != header.Unset {
			fieldOffset, ok := fieldMap[header.Names[tokenCounter]]
			if ok {
				parseTSVField(
					lineString[:tokenEndIdx],
					header.Types[tokenCounter],
					data.Field(fieldOffset),
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
		fieldOffset, ok := fieldMap[header.Names[tokenCounter]]
		if ok {
			parseTSVField(
				lineString,
				header.Types[tokenCounter],
				data.Field(fieldOffset),
				logger,
			)
		}
	}

	return dat
}
