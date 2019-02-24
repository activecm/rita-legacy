package blacklist

import (
	"strconv"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for host analysis
	analyzer struct {
		chunk            int            //current chunk (0 if not on rolling analysis)
		chunkStr         string         //current chunk (0 if not on rolling analysis)
		db               *database.DB   // provides access to MongoDB
		conf             *config.Config // contains details needed to access MongoDB
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan hostRes   // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for gathering data
func newAnalyzer(chunk int, db *database.DB, conf *config.Config, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
		db:               db,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan hostRes),
	}
}

//collect sends a chunk of data to be analyzed
func (a *analyzer) collect(data hostRes) {
	a.analysisChannel <- data
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

		for data := range a.analysisChannel {

			ip := data.IP
			connectedSrcHosts := a.getBlacklistedIPConnections(ip, "dst", "src")
			connectedDstHosts := a.getBlacklistedIPConnections(ip, "src", "dst")

			for _, entry := range connectedSrcHosts {
				var res3 []hostRes
				newblsrc := false

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(bson.M{"ip": entry.Host, "dat.bl": ip}).All(&res3)

				if !(len(res3) > 0) {
					newblsrc = true
					// fmt.Println("host no results", res3, ip)
				}

				blsrcOutput := hasBlacklistedDstQuery(a.chunk, ip, entry, newblsrc)
				// set to writer channel
				a.analyzedCallback(blsrcOutput)
			}
			for _, entry := range connectedDstHosts {
				var res3 []hostRes
				newbldst := false

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(bson.M{"ip": entry.Host, "dat.bl": ip}).All(&res3)

				if !(len(res3) > 0) {
					newbldst = true
					// fmt.Println("host no results", res3, ip)
				}

				bldstOutput := hasBlacklistedSrcQuery(a.chunk, ip, entry, newbldst)
				// set to writer channel
				a.analyzedCallback(bldstOutput)

			}

		}

		a.analysisWg.Done()
	}()
}

//hasBlacklistedQuery ...
// If the internal system initiated the connection, then bl_out_count
// holds the number of unique blacklisted IPs the given host contacted.
func hasBlacklistedDstQuery(chunk int, ip string, entry uconnRes, newFlag bool) update {

	var output update

	// create query
	query := bson.M{}

	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"bl":             ip,
				"bl_out_count":   1,
				"bl_total_bytes": entry.TotalBytes,
				"bl_conn_count":  entry.Connections,
				"cid":            chunk,
			}}

		// create selector for output
		output.query = query
		output.selector = bson.M{"ip": entry.Host}

	} else {

		query["$set"] = bson.M{
			"dat.$.bl_conn_count":  entry.Connections,
			"dat.$.bl_total_bytes": entry.TotalBytes,
			"dat.$.bl_out_count":   1,
			"dat.$.cid":            chunk,
		}

		// create selector for output
		output.query = query
		output.selector = bson.M{"ip": entry.Host, "dat.bl": ip}
	}

	return output
}

//hasBlacklistedQuery ...
// If the internal system initiated the connection, then bl_out_count
// holds the number of unique blacklisted IPs the given host contacted.
func hasBlacklistedSrcQuery(chunk int, ip string, entry uconnRes, newFlag bool) update {

	var output update

	// create query
	query := bson.M{}

	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"bl":             ip,
				"bl_in_count":    1,
				"bl_total_bytes": entry.TotalBytes,
				"bl_conn_count":  entry.Connections,
				"cid":            chunk,
			}}

		// create selector for output
		output.query = query
		output.selector = bson.M{"ip": entry.Host}

	} else {

		query["$set"] = bson.M{
			"dat.$.bl_conn_count":  entry.Connections,
			"dat.$.bl_total_bytes": entry.TotalBytes,
			"dat.$.bl_in_count":    1,
			"dat.$.cid":            chunk,
		}

		// create selector for output
		output.query = query
		output.selector = bson.M{"ip": entry.Host, "dat.cid": chunk}
	}

	return output
}

// getBlaclistedIPConnections
func (a *analyzer) getBlacklistedIPConnections(ip string, field1 string, field2 string) []uconnRes {
	ssn := a.db.Session.Copy()
	defer ssn.Close()

	var blIPs []uconnRes

	blIPQuery := []bson.M{
		bson.M{"$match": bson.M{field1: ip}},
		bson.M{"$unwind": "$dat"},
		bson.M{"$group": bson.M{
			"_id":            "$" + field2,
			"bl_conn_count":  bson.M{"$sum": "$dat.count"},
			"bl_in_count":    bson.M{"$sum": 1},
			"bl_total_bytes": bson.M{"$sum": "$dat.tbytes"},
		}},
	}

	_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.UniqueConnTable).Pipe(blIPQuery).All(&blIPs)

	return blIPs

}
