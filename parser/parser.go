package parser

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bglebrun/rita/database"
)

type (
	ParsedLine interface {
		IsWhiteListed(whitelist []string) bool
		TargetCollection() string
	}

	creatorFunc   func() ParsedLine // A function that creates arbitrary objects
	processorFunc func(ParsedLine)  // A function that processes arbitrary objects

	docParser struct { // The document parsing structure
		file      *database.IndexedFile // the file we are parsing
		writer    *DocWriter            // writer to write out the records
		Errors    []error               // All errors for this file
		log       *log.Logger           // log output
		curFile   *os.File              // currently open file
		creator   creatorFunc           // For creating the objects
		processor processorFunc         // For processing the objects (may be nil)
		unParsed  chan string           // Records for parsing
		SFields   map[string]int        // A field lookup for types
		Header    struct {              // Header maintains the header of the bro log
			Names     []string // Names of fields
			Types     []string // Types of fields
			Separator string   // Field separator
			SetSep    string   // Set separator
			Empty     string   // Empty field tag
			Unset     string   // Unset field tag
			ObjType   string   // Object type (comes from #path)
		}
	}
)

// ParseFile generates a document parser and parses the file to the writer
// Pass this a started writer. Otherwise the writers will be started several times and may lock
// out unexpectedly.
func parseFile(file *database.IndexedFile, wr *DocWriter, res *database.Resources) error {
	d := &docParser{file: file, writer: wr}
	d.log = res.Log
	d.unParsed = make(chan string, 100)
	d.SFields = make(map[string]int)

	scn, err := d.getScanner()
	if err != nil {
		d.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("parser exiting early")
		return err
	}

	err = d.scanHeader(scn)
	if err != nil {
		d.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("scanHeader failure, exiting early")
		return err
	}
	d.curFile.Close()

	scn, err = d.getScanner()
	if err != nil {
		d.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("parser exiting early (getScanner)")
		return err
	}

	//This error is reported as info since it simply means we don't
	//have logic for this log type yet
	err = d.setStructType()
	if err != nil {
		d.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Info("parser exiting early (setStructType)")
		return err
	}

	err = d.validateStruct(d.creator())
	if err != nil {
		d.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("exiting early (validateStruct)")
		return err
	}

	wg := new(sync.WaitGroup)
	for i := 0; i < d.writer.threadCount; i++ {
		wg.Add(1)
		go func() {
			d.parseLine()
			wg.Done()
		}()
	}

	for scn.Scan() {
		d.unParsed <- scn.Text()
	}
	if scn.Err() != nil {
		d.log.WithFields(log.Fields{
			"error": scn.Err().Error(),
		}).Error("scanner encountered an error")
	}
	close(d.unParsed)

	wg.Wait()
	return nil
}

// parseLine ... you can probably guess
func (d *docParser) parseLine() {

	d.log.Debug("Started line parser")
	for ln := range d.unParsed {

		d.log.Debug("parsing line ", ln)
		line := strings.Split(ln, string("\x09"))
		if len(line) < 1 {
			continue
		}
		if strings.Contains(line[0], "#") {
			continue
		}
		dat := d.creator()
		data := reflect.ValueOf(dat).Elem()

		if len(line) < len(d.Header.Names) {
			d.log.WithFields(log.Fields{
				"line": ln,
			}).Error("Mismatched column count in parsed line.")
			continue
		}

		for idx, val := range d.Header.Names {
			if line[idx] == d.Header.Empty ||
				line[idx] == d.Header.Unset {
				continue
			}

			switch d.Header.Types[idx] {
			case TIME:
				secs := strings.Split(line[idx], ".")
				s, err := strconv.ParseInt(secs[0], 10, 64)
				if err != nil {
					d.log.WithFields(log.Fields{
						"error": err.Error(),
						"value": line[idx],
					}).Error("Couldn't convert unix ts")
					data.Field(d.SFields[val]).SetInt(-1)
					break
				}

				n, err := strconv.ParseInt(secs[1], 10, 64)
				if err != nil {
					d.log.WithFields(log.Fields{
						"error": err.Error(),
						"value": line[idx],
					}).Error("Couldn't convert unix ts")
					data.Field(d.SFields[val]).SetInt(-1)
					break
				}

				ttim := time.Unix(s, n)
				tval := ttim.Unix()
				data.Field(d.SFields[val]).SetInt(tval)
				d.file.Date = fmt.Sprintf("%d-%02d-%02d",
					ttim.Year(), ttim.Month(), ttim.Day())
				break
			case STRING:
				data.Field(d.SFields[val]).SetString(line[idx])
				break
			case ADDR:
				data.Field(d.SFields[val]).SetString(line[idx])
				break
			case PORT:
				pval, err := strconv.ParseInt(line[idx], 10, 32)
				if err != nil {
					d.log.WithFields(log.Fields{
						"error": err.Error(),
						"value": line[idx],
					}).Error("Couldn't convert port number")
					data.Field(d.SFields[val]).SetInt(-1)
					break
				}
				data.Field(d.SFields[val]).SetInt(pval)
				break
			case ENUM:
				data.Field(d.SFields[val]).SetString(line[idx])
				break
			case INTERVAL:
				flt, err := strconv.ParseFloat(line[idx], 64)
				if err != nil {
					d.log.WithFields(log.Fields{
						"error": err.Error(),
						"value": line[idx],
					}).Error("Couldn't convert float")
					data.Field(d.SFields[val]).SetFloat(-1.0)
					break
				}
				data.Field(d.SFields[val]).SetFloat(flt)
				break
			case COUNT:
				cnt, err := strconv.ParseInt(line[idx], 10, 64)
				if err != nil {
					d.log.WithFields(log.Fields{
						"error": err.Error(),
						"value": line[idx],
					}).Error("Couldn't convert count")
					data.Field(d.SFields[val]).SetInt(-1)
					break
				}
				data.Field(d.SFields[val]).SetInt(cnt)
				break
			case BOOL:
				if line[idx] == "T" {
					data.Field(d.SFields[val]).SetBool(true)
					break
				}
				data.Field(d.SFields[val]).SetBool(false)
				break
			case "set[string]":
				data.Field(d.SFields[val]).SetString(line[idx])
				break
			case "set[enum]":
				data.Field(d.SFields[val]).SetString(line[idx])
				break
			case "vector[string]":
				data.Field(d.SFields[val]).SetString(line[idx])
				break
			case "vector[duration]":
				data.Field(d.SFields[val]).SetString(line[idx])
				break
			case "vector[interval]":
				data.Field(d.SFields[val]).SetString(line[idx])
				break
			default:
				d.log.WithFields(log.Fields{
					"error": "Unhandled type",
					"value": d.Header.Types[idx],
				}).Error("Encountered unhandled type in log")
			}

		}

		// If the log line needs any alterations before it heads to the db, process it
		if d.processor != nil {
			d.processor(dat)
		}

		//TODO: get Coll from the config
		var toWrite = new(WriteQueuedLine)
		toWrite.line = dat
		toWrite.file = d.file
		d.writer.Write(toWrite)
	}
}

// scanHeader looks through the header of a file to determine the files configuration
func (d *docParser) scanHeader(scan *bufio.Scanner) error {
	d.log.WithFields(log.Fields{
		"path": d.file.Path,
	}).Debug("Entered scanHeader")

	for scan.Scan() {
		if scan.Err() != nil {
			break
		}
		if len(scan.Text()) < 1 {
			continue
		}
		if scan.Text()[0] == '#' {
			d.log.WithFields(log.Fields{
				"line": scan.Text(),
			}).Debug("Found comment line")
			line := strings.Fields(scan.Text())
			if len(line) < 2 {
				continue
			}
			if strings.Contains(line[0], "separator") {
				if line[1] == "\x09" {
					d.Header.Separator = "\t"
				}
			} else if strings.Contains(line[0], "set_separator") {
				d.Header.SetSep = line[1]
			} else if strings.Contains(line[0], "empty_field") {
				d.Header.Empty = line[1]
			} else if strings.Contains(line[0], "unset_field") {
				d.Header.Unset = line[1]
			} else if strings.Contains(line[0], "fields") {
				d.Header.Names = line[1:]
			} else if strings.Contains(line[0], "types") {
				d.Header.Types = line[1:]
			} else if strings.Contains(line[0], "path") {
				d.Header.ObjType = line[1]
			} else {
				continue
			}
		} else {
			break
		}
	}
	if len(d.Header.Names) != len(d.Header.Types) {
		d.log.WithFields(log.Fields{
			"names_len": len(d.Header.Names),
			"types_len": len(d.Header.Types),
		}).Error("Unmatched arrays Names | Types)")

		return errors.New("Name / Type mismatch")
	}

	return nil
}

// getScanner returns a scanner given the path in the object
func (d *docParser) getScanner() (*bufio.Scanner, error) {
	ftype := d.file.Path[len(d.file.Path)-3:]
	if ftype != ".gz" && ftype != "log" {
		err := errors.New("Filetype not recognized")
		d.log.WithFields(log.Fields{
			"error": err.Error(),
			"path":  d.file.Path,
		}).Error("Filetype must be .gz or .log")
		d.Errors = append(d.Errors, err)
		return nil, err
	}

	f, err := os.Open(d.file.Path)
	if err != nil {
		d.log.WithFields(log.Fields{
			"error": err.Error(),
			"path":  d.file.Path,
		}).Error("Couldn't open file")
		d.Errors = append(d.Errors, err)
		return nil, err
	}
	d.curFile = f

	// We have an extra step to do if there was a .gz extension
	if ftype == ".gz" {
		rdr, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			d.log.WithFields(log.Fields{
				"error": err.Error(),
				"path":  d.file.Path,
			}).Error("Couldn't create gzip reader")
			d.Errors = append(d.Errors, err)
			return nil, err
		}
		return bufio.NewScanner(rdr), nil
	}
	// Else (not a gz)
	return bufio.NewScanner(f), nil
}

// validateStruct checks that the fields are correctly typed for the struct we're
// trying to sync this data with. There are several requirements for the bro struct
// that is being passed in.
// 1. The struct must contain an equal number of fields and types (each bro field must
//    have a bro type)
// 2. The struct must not have fewer bro labled fields than the header has defined
// 3. The struct must have a matching field, with matching type for each element in
//    the bro file's header.
func (d *docParser) validateStruct(s interface{}) error {

	// The lookup struct gives us a way to walk the data structure only once
	type lookup struct {
		brotype string
		offset  int
	}

	// map the bro names -> the brotypes
	fields := make(map[string]lookup)

	// this is the internal mapping of the name to is offset in the structure
	d.SFields = make(map[string]int)
	st := reflect.TypeOf(s).Elem()

	// Walk the fields of the given structure
	// fullfills 1 above
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)                // Get the current fields
		bro := f.Tag.Get("bro")         // Get its bro tag
		brotype := f.Tag.Get("brotype") // Get its type tag

		// Check if this is not a bro field
		if len(bro) == 0 && len(brotype) == 0 {
			continue
		}

		// Check that we have a pair of fields
		if len(bro) == 0 || len(brotype) == 0 {
			errval := errors.New("incomplete bro variable")
			d.log.WithFields(log.Fields{
				"path":  d.file.Path,
				"error": errval.Error(),
			}).Error("found an incomplete type: (bro)", bro, "(brotype)",
				brotype, "both fields must be filled in or neither")
			return errval
		}
		lu := lookup{brotype: brotype, offset: i}
		fields[bro] = lu
	}

	// Now walk our fields array and link each field up with a type in our
	// structure. Completes 2 & 3.
	for x, v := range d.Header.Names {

		// Grab the field out of our map, if it's not present that's an error
		field, ok := fields[v]
		if !ok {
			errval := errors.New("unmatched field in log")
			d.log.WithFields(log.Fields{
				"path":          d.file.Path,
				"error":         errval.Error(),
				"missing_field": v,
			}).Error("the log contains a field with no candidate in the data structure")
			return errval
		}

		// Check to ensure that the fields in the file have the same type as in the data structure
		if d.Header.Types[x] != field.brotype {
			errval := errors.New("Type mismatch found in log")
			d.log.WithFields(log.Fields{
				"path":            d.file.Path,
				"error":           errval.Error(),
				"field_name":      v,
				"log_has_type":    d.Header.Types[x],
				"struct_has_type": field.brotype,
			}).Error("the types in the log must exactly match the brotype tags of the struct")
			return errval
		}

		// We've validated this field and type map it in the header file
		d.SFields[v] = field.offset
	}

	// Having completed both loops with no errors we're safe to move on
	return nil
}

// setStructType sets the structure type that we want for a line parser
// this is done by taking the path variable out of the header and using it
// as a lookup for mapping a creatorFunc to the object
func (d *docParser) setStructType() error {
	switch d.Header.ObjType {
	case "conn":
		d.creator = func() ParsedLine {
			return &Conn{}
		}
		break
	case "dns":
		d.creator = func() ParsedLine {
			return &DNS{}
		}
		break
	case "http":
		d.creator = func() ParsedLine {
			return &HTTP{}
		}
		d.processor = processHTTP // fixes absolute vs relative uris
		break
	default:
		return errors.New("Unknown log type")
	}
	return nil
}
