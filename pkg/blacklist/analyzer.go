package blacklist

import (
	"strconv"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for host analysis
	analyzer struct {
		chunk            int                //current chunk (0 if not on rolling analysis)
		chunkStr         string             //current chunk (0 if not on rolling analysis)
		db               *database.DB       // provides access to MongoDB
		conf             *config.Config     // contains details needed to access MongoDB
		analyzedCallback func(hostsUpdate)  // called on each analyzed result
		closedCallback   func()             // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan data.UniqueIP // holds unanalyzed data
		analysisWg       sync.WaitGroup     // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for gathering data
func newAnalyzer(chunk int, db *database.DB, conf *config.Config, analyzedCallback func(hostsUpdate), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
		db:               db,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan data.UniqueIP),
	}
}

//collect sends a chunk of data to be analyzed
func (a *analyzer) collect(datum data.UniqueIP) {
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

		for blacklistedIP := range a.analysisChannel {
			blDstUconns := a.getUniqueConnsforBLDestination(blacklistedIP)
			blSrcUconns := a.getUniqueConnsforBLSource(blacklistedIP)

			for _, blUconnData := range blDstUconns { // update sources which contacted the blacklisted destination
				var blHostEntries []data.UniqueIP
				newBLSrc := false

				srcBLKey := blUconnData.Host.BSONKey()
				srcBLKey["dat.bl"] = blacklistedIP.BSONKey()

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(srcBLKey).All(&blHostEntries)

				if !(len(blHostEntries) > 0) {
					newBLSrc = true
					// fmt.Println("host no results", blHostEntries, blacklistedIP)
				}

				srcHostUpdate := appendBlacklistedDstQuery(a.chunk, blacklistedIP, blUconnData, newBLSrc)

				// set to writer channel
				a.analyzedCallback(srcHostUpdate)
			}
			for _, blUconnData := range blSrcUconns { // update destinations which were contacted by the blacklisted source
				var blHostEntries []data.UniqueIP
				newBLDst := false

				dstBLKey := blUconnData.Host.BSONKey()
				dstBLKey["dat.bl"] = blacklistedIP.BSONKey()

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(dstBLKey).All(&blHostEntries)

				if !(len(blHostEntries) > 0) {
					newBLDst = true
					// fmt.Println("host no results", blHostEntries, ip)
				}

				dstHostUpdate := appendBlacklistedSrcQuery(a.chunk, blacklistedIP, blUconnData, newBLDst)
				// set to writer channel
				a.analyzedCallback(dstHostUpdate)

			}

		}

		a.analysisWg.Done()
	}()
}

//appendBlacklistedDstQuery adds a blacklist record to a host which contacted by a blacklisted destination
func appendBlacklistedDstQuery(chunk int, blacklistedDst data.UniqueIP, srcConnData connectionPeer, newFlag bool) hostsUpdate {
	var output hostsUpdate

	// create query
	query := bson.M{}

	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"bl":             blacklistedDst,
				"bl_out_count":   1,
				"bl_total_bytes": srcConnData.TotalBytes,
				"bl_conn_count":  srcConnData.Connections,
				"cid":            chunk,
			}}

		// create selector for output
		output.query = query
		output.selector = srcConnData.Host.BSONKey()

	} else {

		query["$set"] = bson.M{
			"dat.$.bl_conn_count":  srcConnData.Connections,
			"dat.$.bl_total_bytes": srcConnData.TotalBytes,
			"dat.$.bl_out_count":   1,
			"dat.$.cid":            chunk,
		}

		// create selector for output
		output.selector = srcConnData.Host.BSONKey()
		output.selector["dat.bl"] = blacklistedDst.BSONKey()
	}

	return output
}

//appendBlacklistedSrcQuery adds a blacklist record to a host which was contacted by a blacklisted source
func appendBlacklistedSrcQuery(chunk int, blacklistedSrc data.UniqueIP, dstConnData connectionPeer, newFlag bool) hostsUpdate {
	var output hostsUpdate

	// create query
	query := bson.M{}

	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"bl":             blacklistedSrc,
				"bl_in_count":    1,
				"bl_total_bytes": dstConnData.TotalBytes,
				"bl_conn_count":  dstConnData.Connections,
				"cid":            chunk,
			}}

		// create selector for output
		output.query = query
		output.selector = dstConnData.Host.BSONKey()

	} else {

		query["$set"] = bson.M{
			"dat.$.bl_conn_count":  dstConnData.Connections,
			"dat.$.bl_total_bytes": dstConnData.TotalBytes,
			"dat.$.bl_in_count":    1,
			"dat.$.cid":            chunk,
		}

		// create selector for output
		output.selector = dstConnData.Host.BSONKey()
		output.selector["dat.bl"] = blacklistedSrc.BSONKey()
	}

	return output
}

//getUniqueConnsforBLDestination returns the IP addresses that contacted a given blacklisted IP along with the number
//of connections and bytes sent
func (a *analyzer) getUniqueConnsforBLDestination(blDestinationIP data.UniqueIP) []connectionPeer {
	ssn := a.db.Session.Copy()
	defer ssn.Close()

	var blIPs []connectionPeer

	blIPQuery := []bson.M{
		{"$match": bson.M{
			"dst":              blDestinationIP.IP,
			"dst_network_uuid": blDestinationIP.NetworkUUID,
		}},
		{"$unwind": "$dat"},
		{"$group": bson.M{
			"_id": bson.M{
				"ip":           "$src",
				"network_uuid": "$src_network_uuid",
			},
			"bl_conn_count":  bson.M{"$sum": "$dat.count"},
			"bl_total_bytes": bson.M{"$sum": "$dat.tbytes"},
			// I don't think that either of these fields will ever be more than one value...
			// ...but just in case
			"open_bytes":            bson.M{"$sum": "$open_bytes"},
			"open_connection_count": bson.M{"$sum": "$open_connection_count"},
		}},
		{"$project": bson.M{
			"_id":            1,
			"bl_conn_count":  bson.M{"$sum": []interface{}{"$bl_conn_count", "$open_connection_count"}},
			"bl_total_bytes": bson.M{"$sum": []interface{}{"$bl_total_bytes", "$open_bytes"}},
		}},
	}

	_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.UniqueConnTable).Pipe(blIPQuery).AllowDiskUse().All(&blIPs)

	return blIPs
}

//getUniqueConnsforBLSource returns the IP addresses that a given blacklisted IP contacted along with the number
//of connections and bytes sent
func (a *analyzer) getUniqueConnsforBLSource(blSourceIP data.UniqueIP) []connectionPeer {
	ssn := a.db.Session.Copy()
	defer ssn.Close()

	var blIPs []connectionPeer

	blIPQuery := []bson.M{
		{"$match": bson.M{
			"src":              blSourceIP.IP,
			"src_network_uuid": blSourceIP.NetworkUUID,
		}},
		{"$unwind": "$dat"},
		{"$group": bson.M{
			"_id": bson.M{
				"ip":           "$dst",
				"network_uuid": "$dst_network_uuid",
			},
			"bl_conn_count":         bson.M{"$sum": "$dat.count"},
			"bl_total_bytes":        bson.M{"$sum": "$dat.tbytes"},
			"open_bytes":            bson.M{"$sum": "$open_bytes"},
			"open_connection_count": bson.M{"$sum": "$open_connection_count"},
		}},
		{"$project": bson.M{
			"_id":            1,
			"bl_conn_count":  bson.M{"$sum": []interface{}{"$bl_conn_count", "$open_connection_count"}},
			"bl_total_bytes": bson.M{"$sum": []interface{}{"$bl_total_bytes", "$open_bytes"}},
		}},
	}

	_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.UniqueConnTable).Pipe(blIPQuery).AllowDiskUse().All(&blIPs)

	return blIPs
}
