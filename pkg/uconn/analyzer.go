package uconn

import (
	"strconv"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for exploded dns analysis
	analyzer struct {
		chunk            int            //current chunk (0 if not on rolling analysis)
		chunkStr         string         //current chunk (0 if not on rolling analysis)
		connLimit        int64          // limit for strobe classification
		db               *database.DB   // provides access to MongoDB
		conf             *config.Config // contains details needed to access MongoDB
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Pair     // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for parsing hostnames
func newAnalyzer(chunk int, connLimit int64, db *database.DB, conf *config.Config, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
		connLimit:        connLimit,
		db:               db,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *Pair),
	}
}

//collect sends a group of domains to be analyzed
func (a *analyzer) collect(data *Pair) {
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

		for data := range a.analysisChannel {
			// set up writer output
			var output update

			// create query
			query := bson.M{
				"$set": bson.M{
					"local_src": data.IsLocalSrc,
					"local_dst": data.IsLocalDst,
				},
			}

			if len(data.Tuples) > 5 {
				data.Tuples = data.Tuples[:5]
			}

			// if this connection qualifies to be a strobe with the current number
			// of connections in the currently parsing in data, don't store bytes and ts.
			// it will not qualify to be downgraded to a beacon until this chunk is
			// outdated and removed. If only importing once - still just a strobe.
			if data.ConnectionCount >= a.connLimit {
				query["$set"] = bson.M{"strobe": true, "cid": a.chunk}
				query["$push"] = bson.M{
					"dat": bson.M{
						"count":  data.ConnectionCount,
						"bytes":  []interface{}{},
						"ts":     []interface{}{},
						"tuples": data.Tuples,
						"icerts": data.InvalidCertFlag,
						"maxdur": data.MaxDuration,
						"tbytes": data.TotalBytes,
						"tdur":   data.TotalDuration,
						"cid":    a.chunk,
					},
				}
			} else {
				query["$push"] = bson.M{
					"dat": bson.M{
						"count":  data.ConnectionCount,
						"bytes":  data.OrigBytesList,
						"ts":     data.TsList,
						"tuples": data.Tuples,
						"icerts": data.InvalidCertFlag,
						"maxdur": data.MaxDuration,
						"tbytes": data.TotalBytes,
						"tdur":   data.TotalDuration,
						"cid":    a.chunk,
					},
				}
				query["$set"] = bson.M{"cid": a.chunk}
			}

			// assign formatted query to output
			output.uconn.query = query

			//TODO[AGENT]: Change selector to use UniqueIP's NetworkID
			// create selector for output
			output.uconn.selector = bson.M{"src": data.Src, "dst": data.Dst}

			// get maxdur host table update
			// since we are only updating stats for internal ips (as defined by the
			// user in the file), we need to customize the query to update based on
			// which ip in the connection was local.
			if data.IsLocalSrc == true {
				output.hostMaxDur = a.hostMaxDurQuery(data.MaxDuration, data.Src, data.Dst)
			} else if data.IsLocalDst {
				output.hostMaxDur = a.hostMaxDurQuery(data.MaxDuration, data.Dst, data.Src)
			}

			// set to writer channel
			a.analyzedCallback(output)

		}
		a.analysisWg.Done()
	}()
}

//TODO[AGENT]: Change externalIP to NetworkID in hostMaxDurQuery
func (a *analyzer) hostMaxDurQuery(maxDur float64, localIP string, externalIP string) updateInfo {
	ssn := a.db.Session.Copy()
	defer ssn.Close()

	var output updateInfo

	// create query
	query := bson.M{}

	// check if we need to update
	// we do this before the other queries because otherwise if a max dur
	// starts out with a high number which reduces over time, it will keep
	// the incorrect high max for that specific destination.
	var resListExactMatch []interface{}

	//TODO[AGENT]: Change mdip to use UniqueIP's NetworkID
	maxDurMatchExactQuery := bson.M{
		"ip":  localIP,
		"dat": bson.M{"$elemMatch": bson.M{"mdip": externalIP, "max_duration": bson.M{"$lte": maxDur}}},
	}

	_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(maxDurMatchExactQuery).All(&resListExactMatch)

	// if we have exact matches, update to new score and return
	if len(resListExactMatch) > 0 {

		// update chunk number
		query["$set"] = bson.M{
			"dat.$.cid":          a.chunk,
			"dat.$.max_duration": maxDur,
		}

		// create selector for output
		output.query = query

		// using the same find query we created above will allow us to match and
		// update the exact chunk we need to update
		output.selector = maxDurMatchExactQuery

		return output
	}

	// The below is only for cases where the ip is not currently listed as a max dur
	// for a source
	// update max dur
	newFlag := false
	updateFlag := false

	var resListLower []interface{}
	var resListUpper []interface{}

	//TODO[AGENT]: Change ip to use UniqueIP's NetworkID in host table update/ queries

	// this query will find any matching chunk that is reporting a lower
	// max beacon score than the current one we are working with
	maxDurMatchLowerQuery := bson.M{
		"ip": localIP,
		"dat": bson.M{"$elemMatch": bson.M{
			"cid":          a.chunk,
			"max_duration": bson.M{"$lte": maxDur},
		}},
	}

	maxDurMatchUpperQuery := bson.M{
		"ip": localIP,
		"dat": bson.M{"$elemMatch": bson.M{
			"cid":          a.chunk,
			"max_duration": bson.M{"$gte": maxDur},
		}},
	}

	// find matching lower chunks
	_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(maxDurMatchLowerQuery).All(&resListLower)

	// if no matching chunks are found, we will set the new flag
	if !(len(resListLower) > 0) {

		// find matching upper chunks
		_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(maxDurMatchUpperQuery).All(&resListUpper)

		// update if no upper chunks are found
		if !(len(resListUpper) > 0) {
			newFlag = true
		}
	} else {
		updateFlag = true
	}

	// since we didn't find any changeable lower max duration scores, we will
	// set the condition to push a new entry with the current score listed as the
	// max beacon ONLY if no matching chunks reporting higher max beacon scores
	// are found.

	//TODO[AGENT]: Change mdip to use UniqueIP's NetworkID
	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"max_duration": maxDur,
				"mdip":         externalIP,
				"cid":          a.chunk,
			}}

		// create selector for output
		output.query = query
		output.selector = bson.M{"ip": localIP}

	} else if updateFlag {
		query["$set"] = bson.M{
			"dat.$.max_duration": maxDur,
			"dat.$.mdip":         externalIP,
			"dat.$.cid":          a.chunk,
		}

		// create selector for output
		output.query = query

		// using the same find query we created above will allow us to match and
		// update the exact chunk we need to update
		output.selector = maxDurMatchLowerQuery
	}

	return output
}
