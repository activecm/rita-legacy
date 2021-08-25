package useragent

import (
	"strconv"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// rareSignatureOrigIPsCutoff determines the cutoff for marking a particular IP as having used
// rare signature on an HTTP(s) connection. If a particular signature/ user agent is associated
// with less than `rareSignatureOrigIPsCutoff` originating IPs, we mark those IPs as having used
// a rare signature.
const rareSignatureOrigIPsCutoff = 5

type (
	//analyzer is a structure for useragent analysis
	analyzer struct {
		chunk            int            //current chunk (0 if not on rolling analysis)
		chunkStr         string         //current chunk (0 if not on rolling analysis)
		db               *database.DB   // provides access to MongoDB
		conf             *config.Config // contains details needed to access MongoDB
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input    // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for parsing hostnames
func newAnalyzer(chunk int, db *database.DB, conf *config.Config, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
		db:               db,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *Input),
	}
}

//collect sends a group of domains to be analyzed
func (a *analyzer) collect(datum *Input) {
	a.analysisChannel <- datum
}

//close waits for the collector to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

//start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {
		ssn := a.db.Session.Copy()
		defer ssn.Close()

		for datum := range a.analysisChannel {
			// first we update the useragent collection
			a.updateUseragentCollection(ssn, datum)

			// next we update the hosts collection with rarely used j3 and useragent hosts
			a.updateHostsCollection(ssn, datum)
		}

		a.analysisWg.Done()
	}()
}

// updateUseragentCollection inserts the given datum into the useragent collection. The useragent's
// originating IPs and requested FQDNs are capped in order to prevent hitting the MongoDB document size limits.
func (a *analyzer) updateUseragentCollection(ssn *mgo.Session, datum *Input) {
	// set up writer output
	var output update

	origIPs := datum.OrigIps.Items()
	if len(origIPs) > 10 {
		origIPs = origIPs[:10]
	}

	requests := datum.Requests.Items()
	if len(requests) > 10 {
		requests = requests[:10]
	}

	// create query
	query := bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"seen":     datum.Seen,
				"orig_ips": origIPs,
				"hosts":    requests,
				"cid":      a.chunk,
			},
		},
		"$set":         bson.M{"cid": a.chunk},
		"$setOnInsert": bson.M{"ja3": datum.JA3},
	}

	output.query = query

	output.collection = a.conf.T.UserAgent.UserAgentTable
	// create selector for output
	output.selector = bson.M{"user_agent": datum.Name}

	// set to writer channel
	a.analyzedCallback(output)
}

// updateHostsCollection updates the hosts collection with rarely used j3 and useragent hosts.
// The useragent must have been associated with less than `rareSignatureOrigIPsCutoff` originating IPs
// in order to make it into the hosts collection.
func (a *analyzer) updateHostsCollection(ssn *mgo.Session, datum *Input) {
	// if there are too many IPs associated with this signature in the parsed in files, ignore the
	// rare signature host collection update
	if len(datum.OrigIps) >= rareSignatureOrigIPsCutoff {
		return
	}

	// get the origIPs from the database for the given user agent
	// since the useragent collection update may not have taken place yet, we need to union in
	// the new origIPs from the recently parsed in files
	dbRareSigOrigIPs, _ := a.getOrigIPsForAgentFromDB(ssn, datum.Name)

	// if there are too many IPs associated with this signature in the database, ignore the
	// rare signature host collection update
	if len(dbRareSigOrigIPs) >= rareSignatureOrigIPsCutoff {
		return
	}

	// merge the two lists of origIPs to get rid of duplicates
	origIPsUnioned := unionUniqueIPSlices(datum.OrigIps.Items(), dbRareSigOrigIPs)

	// if we've busted over the limit after unioning the new and old originating IPs together
	// don't update the host records
	if len(origIPsUnioned) >= rareSignatureOrigIPsCutoff {
		return
	}

	// insert host entries for the current parse set
	for _, rareSigIP := range datum.OrigIps {

		newRecordFlag := false // have we created a rare signature entry for this host in this chunk yet?

		entryHostQuery := rareSigIP.BSONKey()
		entryHostQuery["dat.rsig"] = datum.Name

		nExistingEntries, _ := ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(entryHostQuery).Count()

		if nExistingEntries == 0 {
			newRecordFlag = true
		}

		output := hostQuery(a.chunk, datum.Name, rareSigIP, newRecordFlag)
		output.collection = a.conf.T.Structure.HostTable

		// send update to writer channel
		a.analyzedCallback(output)
	}
}

// unionUniqueIPSlices merges two UniqueIP slices into one while removing duplicates.
func unionUniqueIPSlices(slice1 []data.UniqueIP, slice2 []data.UniqueIP) []data.UniqueIP {
	ipsUnionMap := make(map[string]data.UniqueIP)
	for i := range slice1 {
		ipsUnionMap[slice1[i].MapKey()] = slice1[i]
	}
	for i := range slice2 {
		ipsUnionMap[slice2[i].MapKey()] = slice2[i]
	}

	ipsUnionSlice := make([]data.UniqueIP, 0, len(ipsUnionMap))

	for _, ip := range ipsUnionMap {
		ipsUnionSlice = append(ipsUnionSlice, ip)
	}
	return ipsUnionSlice
}

/*
getOrigIPsForAgentFromDB returns the originating IPs associated witha given useragent from the database

db.getCollection('useragent').aggregate([
    {"$match": {"user_agent": "Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.16299.248"}},
    {"$project": {"ips": "$dat.orig_ips"}},
    {"$unwind": "$ips"},
    {"$unwind": "$ips"},
    {"$group": {
            "_id": {
                    "ip" : "$ips.ip",
                    "network_uuid": "$ips.network_uuid",
            },
            "network_name": {"$last": "$ips.network_name"},
    }},
    {"$project": {
		"_id": 0,
        "ip":           "$_id.ip",
        "network_uuid": "$_id.network_uuid",
        "network_name": "$network_name",
    }},
])
*/
func (a *analyzer) getOrigIPsForAgentFromDB(dbSession *mgo.Session, name string) ([]data.UniqueIP, error) {
	query := []bson.M{
		{"$match": bson.M{"user_agent": name}},
		{"$project": bson.M{"ips": "$dat.orig_ips"}},
		{"$unwind": "$ips"},
		{"$unwind": "$ips"}, // not an error, needs to be done twice
		{"$group": bson.M{
			"_id": bson.M{
				"ip":           "$ips.ip",
				"network_uuid": "$ips.network_uuid",
			},
			"network_name": bson.M{"$last": "$ips.network_name"},
		}},
		{"$project": bson.M{
			"_id":          0,
			"ip":           "$_id.ip",
			"network_uuid": "$_id.network_uuid",
			"network_name": "$network_name",
		}},
	}

	var dbRareSigOrigIPs []data.UniqueIP

	err := dbSession.DB(a.db.GetSelectedDB()).C(a.conf.T.UserAgent.UserAgentTable).Pipe(query).AllowDiskUse().
		All(&dbRareSigOrigIPs)

	return dbRareSigOrigIPs, err
}

//hostQuery formats a MongoDB update which either inserts a new rare signature host
//record into a host's dat array in the host collection or updates an existing
//record in the host's dat array for the rare signature with the current chunk id.
func hostQuery(chunk int, useragentStr string, ip data.UniqueIP, newFlag bool) update {
	var output update

	// create query
	query := bson.M{}

	if newFlag {
		// add a new rare signature entry
		query["$push"] = bson.M{
			"dat": bson.M{
				"rsig":  useragentStr,
				"rsigc": 1,
				"cid":   chunk,
			}}

		// create selector for output ,
		output.query = query
		output.selector = ip.BSONKey()

	} else {
		// no need to update all of the fields for an existing
		// record; we just need to update the chunk ID
		query["$set"] = bson.M{
			"dat.$.cid": chunk,
		}

		// create selector for output
		// we don't add cid to the selector query because if the useragent string is
		// already listed, we just want to update it to the most recent chunk instead
		// of adding more
		output.query = query
		output.selector = ip.BSONKey()
		output.selector["dat.rsig"] = useragentStr
	}

	return output
}
