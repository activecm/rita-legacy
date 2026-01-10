package blacklist

import (
	"sync"

	"github.com/activecm/rita-legacy/config"
	"github.com/activecm/rita-legacy/database"
	"github.com/activecm/rita-legacy/pkg/data"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	//analyzer is a structure for marking the peers of unsafe hosts in the host collection
	analyzer struct {
		chunk            int                        //current chunk (0 if not on rolling analysis)
		db               *database.DB               // provides access to MongoDB
		conf             *config.Config             // contains details needed to access MongoDB
		log              *log.Logger                // main logger for RITA
		analyzedCallback func(database.BulkChanges) // called on each analyzed result
		closedCallback   func()                     // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan data.UniqueIP         // holds unanalyzed data
		analysisWg       sync.WaitGroup             // wait for analysis to finish
	}
)

// newAnalyzer creates a new analyzer for summarizing connections to unsafe hosts in the host collection
func newAnalyzer(chunk int, db *database.DB, conf *config.Config, log *log.Logger, analyzedCallback func(database.BulkChanges), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		db:               db,
		conf:             conf,
		log:              log,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan data.UniqueIP),
	}
}

// collect gathers hosts that are known to be unsafe as input to the analyzer
func (a *analyzer) collect(datum data.UniqueIP) {
	a.analysisChannel <- datum
}

// close waits for the analyzer to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

// start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {
		ssn := a.db.Session.Copy()
		defer ssn.Close()

		for blacklistedIP := range a.analysisChannel {
			blDstUconns, err := a.getUniqueConnsforBLDestination(blacklistedIP)
			if err != nil {
				a.log.WithFields(log.Fields{
					"Module": "bl_updater",
					"IP":     blacklistedIP,
				}).Error(err)
			}
			blSrcUconns, err := a.getUniqueConnsforBLSource(blacklistedIP)
			if err != nil {
				a.log.WithFields(log.Fields{
					"Module": "bl_updater",
					"IP":     blacklistedIP,
				}).Error(err)
			}

			for _, blUconnData := range blDstUconns { // update sources which contacted the blacklisted destination
				blDstForSrcExists, err := blHostRecordExists(
					ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable), blUconnData.Host, blacklistedIP,
				)
				if err != nil {
					a.log.WithFields(log.Fields{
						"Module": "bl_updater",
						"IP":     blacklistedIP,
					}).Error(err)
				}
				srcHostUpdate := appendBlacklistedDstQuery(
					a.chunk, blacklistedIP, blUconnData, blDstForSrcExists,
				)

				a.analyzedCallback(database.BulkChanges{a.conf.T.Structure.HostTable: []database.BulkChange{srcHostUpdate}})
			}
			for _, blUconnData := range blSrcUconns { // update destinations which were contacted by the blacklisted source
				blSrcForDstExists, err := blHostRecordExists(
					ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable), blUconnData.Host, blacklistedIP,
				)
				if err != nil {
					a.log.WithFields(log.Fields{
						"Module": "bl_updater",
						"IP":     blacklistedIP,
					}).Error(err)
				}

				newBLSrcForDstUpdate := appendBlacklistedSrcQuery(
					a.chunk, blacklistedIP, blUconnData, blSrcForDstExists,
				)
				a.analyzedCallback(database.BulkChanges{a.conf.T.Structure.HostTable: []database.BulkChange{newBLSrcForDstUpdate}})
			}

		}

		a.analysisWg.Done()
	}()
}

// blHostRecordExists checks if a the hostEntryIP has previously been marked as the peer of the given blacklistedIP
func blHostRecordExists(hostCollection *mgo.Collection, hostEntryIP, blacklistedIP data.UniqueIP) (bool, error) {
	entryKey := hostEntryIP.BSONKey()
	entryKey["dat"] = bson.M{"$elemMatch": blacklistedIP.PrefixedBSONKey("bl")}

	nExistingEntries, err := hostCollection.Find(entryKey).Count()

	return nExistingEntries != 0, err
}

// appendBlacklistedDstQuery adds a blacklist record to a host which contacted by a blacklisted destination
func appendBlacklistedDstQuery(chunk int, blacklistedDst data.UniqueIP, srcConnData connectionPeer, existsFlag bool) database.BulkChange {
	var output database.BulkChange
	output.Upsert = true

	// create query
	query := bson.M{}

	if !existsFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"bl":             blacklistedDst,
				"bl_out_count":   1,
				"bl_total_bytes": srcConnData.TotalBytes,
				"bl_conn_count":  srcConnData.Connections,
				"cid":            chunk,
			}}
		output.Update = query

		// create selector for output
		output.Selector = srcConnData.Host.BSONKey()

	} else {

		query["$set"] = bson.M{
			"dat.$.bl_conn_count":  srcConnData.Connections,
			"dat.$.bl_total_bytes": srcConnData.TotalBytes,
			"dat.$.bl_out_count":   1,
			"dat.$.cid":            chunk,
		}
		output.Update = query

		// create selector for output
		output.Selector = database.MergeBSONMaps(
			srcConnData.Host.BSONKey(),
			bson.M{"dat": bson.M{"$elemMatch": blacklistedDst.PrefixedBSONKey("bl")}},
		)
	}

	return output
}

// appendBlacklistedSrcQuery adds a blacklist record to a host which was contacted by a blacklisted source
func appendBlacklistedSrcQuery(chunk int, blacklistedSrc data.UniqueIP, dstConnData connectionPeer, existsFlag bool) database.BulkChange {
	var output database.BulkChange
	output.Upsert = true

	// create query
	query := bson.M{}

	if !existsFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"bl":             blacklistedSrc,
				"bl_in_count":    1,
				"bl_total_bytes": dstConnData.TotalBytes,
				"bl_conn_count":  dstConnData.Connections,
				"cid":            chunk,
			}}
		output.Update = query

		// create selector for output
		output.Selector = dstConnData.Host.BSONKey()

	} else {

		query["$set"] = bson.M{
			"dat.$.bl_conn_count":  dstConnData.Connections,
			"dat.$.bl_total_bytes": dstConnData.TotalBytes,
			"dat.$.bl_in_count":    1,
			"dat.$.cid":            chunk,
		}
		output.Update = query

		// create selector for output
		output.Selector = database.MergeBSONMaps(
			dstConnData.Host.BSONKey(),
			bson.M{"dat": bson.M{"$elemMatch": blacklistedSrc.PrefixedBSONKey("bl")}},
		)
	}

	return output
}

// getUniqueConnsforBLDestination returns the IP addresses that contacted a given blacklisted IP along with the number
// of connections and bytes sent
func (a *analyzer) getUniqueConnsforBLDestination(blDestinationIP data.UniqueIP) ([]connectionPeer, error) {
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
			// ...but just in case. In otherwords, open_bytes and open_connection_count should
			// really only ever show up as a single value. Using sum here just in case there is
			// situation that I didn't think of or encounter while testing that results in
			// either of these values showing up as an array of values. This might not
			// be necessary but I don't think it hurts
			"open_bytes":            bson.M{"$sum": "$open_bytes"},
			"open_connection_count": bson.M{"$sum": "$open_connection_count"},
		}},
		{"$project": bson.M{
			"_id":            1,
			"bl_conn_count":  bson.M{"$sum": []interface{}{"$bl_conn_count", "$open_connection_count"}},
			"bl_total_bytes": bson.M{"$sum": []interface{}{"$bl_total_bytes", "$open_bytes"}},
		}},
	}

	err := ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.UniqueConnTable).Pipe(blIPQuery).AllowDiskUse().All(&blIPs)
	if err == mgo.ErrNotFound {
		err = nil
	}

	return blIPs, err
}

// getUniqueConnsforBLSource returns the IP addresses that a given blacklisted IP contacted along with the number
// of connections and bytes sent
func (a *analyzer) getUniqueConnsforBLSource(blSourceIP data.UniqueIP) ([]connectionPeer, error) {
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

	err := ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.UniqueConnTable).Pipe(blIPQuery).AllowDiskUse().All(&blIPs)
	if err == mgo.ErrNotFound {
		err = nil
	}

	return blIPs, err
}
