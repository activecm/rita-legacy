package intel

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database/inteldb"
	"github.com/ocmdev/rita/datatypes/intel"
	"github.com/ocmdev/rita/util"

	log "github.com/Sirupsen/logrus"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type (

	// IntelHandle provides functionality for dealing with intel gathering
	// and handling database interactions for intel objects
	IntelHandle struct {

		// db is the run database for this data set
		db string

		// hostCol gives us the hosts collection
		hostCol string

		// urlCol gives us the urls collcection
		urlCol string

		// intelDB is the database name of our intelligence database
		intelDB string

		// intelDBHandle gives us a handle to the intelligence database
		intelDBHandle *inteldb.IntelDBHandle

		// session provides a handle to the database session
		session *mgo.Session

		// log gives us a logger
		log *log.Logger

		// NetcatPath is the path to the netcat executable
		NetcatPath string

		// AgeThreshold is a settable that gives us the number of days that
		// elapse before we recheck data
		AgeThreshold float64
	}
)

// PEBadCymruFieldCount is a Parse Error (PE) generated when not enough fields were given
// to the parseCymruLine function.
var PEBadCymruFieldCount = errors.New("Line contained incorrect number of fields")
var CLBadIPAddress = errors.New("CymruLookupRecieved an invalid IP address")

const expectedCymruFields = 7

// NewIntelHandle uses a config.Resources to generate a new intel handle
func NewIntelHandle(conf *config.Resources) *IntelHandle {
	ssn := conf.CopySession()
	return &IntelHandle{
		intelDB:       conf.System.HostIntelDB,
		session:       ssn,
		log:           conf.Log,
		NetcatPath:    conf.System.GNUNetcatPath,
		hostCol:       conf.System.StructureConfig.HostTable,
		urlCol:        conf.System.UrlsConfig.UrlsTable,
		db:            conf.System.DB,
		intelDBHandle: inteldb.NewIntelDBHandle(conf),
		AgeThreshold:  float64(30.0) * float64(24.0),
	}
}

// Run builds onto the HostIntelDB by running all of the current external hosts
func (i *IntelHandle) Run() {
	ssn := i.session.Copy()
	defer ssn.Close()

	cur := ssn.DB(i.db).C(i.hostCol)
	extHosts := cur.Find(bson.M{"local": false}).Iter()

	var doc struct {
		IP string `bson:"ip"`
	}

	addresses := make(map[string]bool)

	for extHosts.Next(&doc) {
		var check data.IntelData
		err := i.intelDBHandle.Find(doc.IP).IntelData(&check)
		if err != nil {
			addresses[doc.IP] = true
			continue
		}

		if time.Since(check.IntelDate).Hours() > i.AgeThreshold {
			addresses[doc.IP] = true
			continue
		}

	}

	cur = ssn.DB(i.db).C(i.urlCol)
	extHosts = cur.Find(nil).Iter()

	for extHosts.Next(&doc) {
		var check data.IntelData
		err := i.intelDBHandle.Find(doc.IP).IntelData(&check)
		if err != nil {
			addresses[doc.IP] = true
			continue
		}

		if time.Since(check.IntelDate).Hours() > i.AgeThreshold {
			addresses[doc.IP] = true
			continue
		}
	}

	var tolookup []string
	for key := range addresses {
		tolookup = append(tolookup, key)
	}
	recon := i.CymruWhoisLookup(tolookup)

	// sometimes cymru returns a multiple copies of one host
	datamap := make(map[string]bool)
	for _, val := range recon {

		if _, ok := datamap[val.IP]; !ok {
			i.intelDBHandle.Write(val)
			datamap[val.IP] = true // marks host as done
		}
	}

	i.intelDBHandle.Close()

}

// CymruWhoisLookup takes a list of ip addresses and does a batch lookup against
// the team cymru lookup server
func (i *IntelHandle) CymruWhoisLookup(addresses []string) []data.IntelData {

	var result []data.IntelData
	netcat := i.NetcatPath
	if len(addresses) == 0 {
		i.log.WithFields(log.Fields{
			"error": "Addresses field length 0",
		}).Warning("CymruWhoisLookup called with 0 addresses")
		return nil
	}

	lookup := "begin\nverbose\n"

	for _, val := range addresses {
		if i.validCymruInput(val) {
			lookup += fmt.Sprintf("%s\n", val)
		}
	}

	lookup += "end\n"

	var out bytes.Buffer

	whoisCmd := exec.Command(netcat, "whois.cymru.com", "43")
	whoisCmd.Stdin = strings.NewReader(lookup)
	whoisCmd.Stdout = &out

	whoisCmd.Run()

	outReader := bytes.NewReader(out.Bytes())
	resScanner := bufio.NewScanner(outReader)
	for resScanner.Scan() {
		dat, err := i.parseCymruLine(resScanner.Text())
		if err == nil && len(dat.IP) > 7 {
			result = append(result, dat)
		}
	}

	return result
}

// validCymruInput validates that what is being looked up is reasonable
func (i *IntelHandle) validCymruInput(address string) bool {
	if !util.IsIP(address) {
		i.log.WithFields(log.Fields{
			"error": "IP address was not valid",
			"IP":    address,
		}).Warning("invalid ip address")
		return false
	}

	if util.RFC1918(address) {
		i.log.WithFields(log.Fields{
			"error": "IP address was valid RFC1918",
			"IP":    address,
		}).Warning("invalid ip address")
		return false
	}
	return true
}

// parseCymruLine parses a line from a cymru response
func (i *IntelHandle) parseCymruLine(line string) (data.IntelData, error) {

	var result data.IntelData

	if bytes.HasPrefix([]byte(line), []byte("Bulk mode;")) {
		return result, nil
	}

	vals := strings.Split(line, "|")
	for idx, val := range vals {
		vals[idx] = strings.TrimSpace(val)
	}

	if len(vals) != expectedCymruFields {
		i.log.WithFields(log.Fields{
			"error":       PEBadCymruFieldCount.Error(),
			"expected":    expectedCymruFields,
			"field_count": len(vals),
		}).Error("bad field count in line: ", line)
		return result, PEBadCymruFieldCount
	}

	// attempt to convert the ASN to an int64
	asn, err := strconv.ParseInt(vals[0], 10, 64)
	if err != nil {

		// If the input actually had an NA asn take care of that
		if strings.Contains(vals[0], "NA") {
			asn = int64(-1)
		} else {

			i.log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("parsing asn")
			result.ASN = int64(-1)
		}
	}

	// assign IP, BGP Prefix, Country Code, and Registry
	result.IP = vals[1]
	result.Prefix = vals[2]
	result.CountryCode = vals[3]
	result.Registry = vals[4]

	// The cymru dates are yyyy-mm-dd
	alloc, err := time.Parse("2006-01-02", vals[5])

	if err != nil {

		if vals[5] != "" {
			i.log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("allocated time parsing error")

		}

		alloc, _ = time.Parse("2006-01-02", "1970-01-01")

	}

	result.Allocated = alloc
	result.ASN = asn

	result.ASName = vals[6]

	return result, nil

}
