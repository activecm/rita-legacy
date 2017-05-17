package parser

import (
	"bufio"
	"compress/gzip"
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	fpt "github.com/ocmdev/rita/parser/fileparsetypes"
	pt "github.com/ocmdev/rita/parser/parsetypes"
)

// getFileScanner returns a buffered file scanner for a bro log file
func getFileScanner(fileHandle *os.File) (*bufio.Scanner, error) {
	ftype := fileHandle.Name()[len(fileHandle.Name())-3:]
	if ftype != ".gz" && ftype != "log" {
		return nil, errors.New("Filetype not recognized")
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
// effect of advancing the fileScanner
func scanHeader(fileScanner *bufio.Scanner) (*fpt.BroHeader, error) {
	toReturn := new(fpt.BroHeader)
	for fileScanner.Scan() {
		if fileScanner.Err() != nil {
			break
		}
		if len(fileScanner.Text()) < 1 {
			continue
		}
		//On the comment lines
		if fileScanner.Text()[0] == '#' {
			line := strings.Fields(fileScanner.Text())
			if strings.Contains(line[0], "separator") {
				//TODO: Verify this works
				if line[1] == "\\x09" {
					toReturn.Separator = "\x09"
				}
			} else if strings.Contains(line[0], "set_separator") {
				toReturn.SetSep = line[1]
			} else if strings.Contains(line[0], "empty_field") {
				toReturn.Empty = line[1]
			} else if strings.Contains(line[0], "unset_field") {
				toReturn.Unset = line[1]
			} else if strings.Contains(line[0], "fields") {
				toReturn.Names = line[1:]
			} else if strings.Contains(line[0], "types") {
				toReturn.Types = line[1:]
			} else if strings.Contains(line[0], "path") {
				toReturn.ObjType = line[1]
			}
		} else {
			//We are done parsing the comments
			break
		}
	}

	if len(toReturn.Names) != len(toReturn.Types) {
		return toReturn, errors.New("Name / Type mismatch")
	}
	return toReturn, nil
}

//mapBroHeaderToParserType checks a parsed BroHeader against
//a BroData struct and returns a mapping from bro field names in the
//bro header to the indexes of the respective fields in the BroData struct
func mapBroHeaderToParserType(header *fpt.BroHeader, broDataFactory func() pt.BroData,
	logger *log.Logger) (fpt.BroHeaderIndexMap, error) {
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
			return nil, errors.New("Type mismatch found in log")
		}
	}

	return toReturn, nil
}

//parseLine parses a line of a bro log with a given broHeader, fieldMap, into
//the BroData created by the broDataFactory
func parseLine(lineString string, header *fpt.BroHeader,
	fieldMap fpt.BroHeaderIndexMap, broDataFactory func() pt.BroData,
	logger *log.Logger) pt.BroData {
	line := strings.Split(lineString, header.Separator)
	if len(line) < len(header.Names) {
		return nil
	}
	if strings.Contains(line[0], "#") {
		return nil
	}

	dat := broDataFactory()
	data := reflect.ValueOf(dat).Elem()

	for idx, val := range header.Names {
		if line[idx] == header.Empty ||
			line[idx] == header.Unset {
			continue
		}

		//fields not in the struct will not be parsed
		fieldOffset, ok := fieldMap[val]
		if !ok {
			continue
		}

		switch header.Types[idx] {
		case pt.Time:
			secs := strings.Split(line[idx], ".")
			s, err := strconv.ParseInt(secs[0], 10, 64)
			if err != nil {
				logger.WithFields(log.Fields{
					"error": err.Error(),
					"value": line[idx],
				}).Error("Couldn't convert unix ts")
				data.Field(fieldOffset).SetInt(-1)
				break
			}

			n, err := strconv.ParseInt(secs[1], 10, 64)
			if err != nil {
				logger.WithFields(log.Fields{
					"error": err.Error(),
					"value": line[idx],
				}).Error("Couldn't convert unix ts")
				data.Field(fieldOffset).SetInt(-1)
				break
			}

			ttim := time.Unix(s, n)
			tval := ttim.Unix()
			data.Field(fieldOffset).SetInt(tval)
			break
		case pt.String:
			data.Field(fieldOffset).SetString(line[idx])
			break
		case pt.Addr:
			data.Field(fieldOffset).SetString(line[idx])
			break
		case pt.Port:
			pval, err := strconv.ParseInt(line[idx], 10, 32)
			if err != nil {
				logger.WithFields(log.Fields{
					"error": err.Error(),
					"value": line[idx],
				}).Error("Couldn't convert port number")
				data.Field(fieldOffset).SetInt(-1)
				break
			}
			data.Field(fieldOffset).SetInt(pval)
			break
		case pt.Enum:
			data.Field(fieldOffset).SetString(line[idx])
			break
		case pt.Interval:
			flt, err := strconv.ParseFloat(line[idx], 64)
			if err != nil {
				logger.WithFields(log.Fields{
					"error": err.Error(),
					"value": line[idx],
				}).Error("Couldn't convert float")
				data.Field(fieldOffset).SetFloat(-1.0)
				break
			}
			data.Field(fieldOffset).SetFloat(flt)
			break
		case pt.Count:
			cnt, err := strconv.ParseInt(line[idx], 10, 64)
			if err != nil {
				logger.WithFields(log.Fields{
					"error": err.Error(),
					"value": line[idx],
				}).Error("Couldn't convert count")
				data.Field(fieldOffset).SetInt(-1)
				break
			}
			data.Field(fieldOffset).SetInt(cnt)
			break
		case pt.Bool:
			if line[idx] == "T" {
				data.Field(fieldOffset).SetBool(true)
				break
			}
			data.Field(fieldOffset).SetBool(false)
			break
		case pt.StringSet:
			tokens := strings.Split(line[idx], ",")
			tVal := reflect.ValueOf(tokens)
			data.Field(fieldOffset).Set(tVal)
			break
		case pt.EnumSet:
			tokens := strings.Split(line[idx], ",")
			tVal := reflect.ValueOf(tokens)
			data.Field(fieldOffset).Set(tVal)
			break
		case pt.StringVector:
			tokens := strings.Split(line[idx], ",")
			tVal := reflect.ValueOf(tokens)
			data.Field(fieldOffset).Set(tVal)
			break
		case pt.IntervalVector:
			tokens := strings.Split(line[idx], ",")
			floats := make([]float64, len(tokens))
			for i, val := range tokens {
				var err error
				floats[i], err = strconv.ParseFloat(val, 64)
				if err != nil {
					logger.WithFields(log.Fields{
						"error": err.Error(),
						"value": val,
					}).Error("Couldn't convert float")
					break
				}
			}
			fVal := reflect.ValueOf(floats)
			data.Field(fieldOffset).Set(fVal)
			break
		default:
			logger.WithFields(log.Fields{
				"error": "Unhandled type",
				"value": header.Types[idx],
			}).Error("Encountered unhandled type in log")
		}
	}
	dat.Normalize()
	return dat
}
