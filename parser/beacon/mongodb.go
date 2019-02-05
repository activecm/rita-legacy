package beacon

import (
	"fmt"
	"runtime"

	"github.com/activecm/rita/parser/uconn"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type repo struct {
	res *resources.Resources
}

//NewMongoRepository create new repository
func NewMongoRepository(res *resources.Resources) Repository {
	return &repo{
		res: res,
	}
}

func (r *repo) CreateIndexes() error {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	collectionName := r.res.Config.T.Beacon.BeaconTable

	// Desired indexes
	indexes := []mgo.Index{
		{Key: []string{"-score"}},
		{Key: []string{"$hashed:src"}},
		{Key: []string{"$hashed:dst"}},
		{Key: []string{"-connection_count"}},
	}
	err := r.res.DB.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

//Upsert loops through every new uconn ....
func (r *repo) Upsert(uconnMap map[string]uconn.Pair) {
	//Find the observation period
	minTime, maxTime := r.findAnalysisPeriod()

	//Create the workers
	writerWorker := newWriter(r.res.Config.T.Beacon.BeaconTable, r.res.DB, r.res.Config)

	analyzerWorker := newAnalyzer(
		r.res.DB,
		r.res.Config,
		minTime,
		maxTime,
		writerWorker.collect,
		writerWorker.close,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		analyzerWorker.start()
		writerWorker.start()
	}

	for _, entry := range uconnMap {

		analyzerWorker.collect(entry)

	}
}

// NOTE: this will be updated with an option for rolling analysis to keep track of the
// 			 timestamp in the metadatabase.
func (r *repo) findAnalysisPeriod() (tsMin int64, tsMax int64) {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	// Structure to store the results of queries
	var res struct {
		Timestamp int64 `bson:"ts_list"`
	}

	// Get min timestamp
	// Build query for aggregation
	tsMinQuery := []bson.M{
		bson.M{"$project": bson.M{"ts_list": 1}}, // include the "ts_list" field from every document
		bson.M{"$unwind": "$ts_list"},            // concat all the lists together
		bson.M{"$sort": bson.M{"ts_list": 1}},    // sort all the timestamps from low to high
		bson.M{"$limit": 1},                      // take the lowest
	}

	// Execute query
	err := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.Structure.UniqueConnTable).Pipe(tsMinQuery).One(&res)

	// Check for errors and parse results
	if err != nil {
		fmt.Println(err)
	} else {
		tsMin = res.Timestamp
	}

	// Get max timestamp
	// Build query for aggregation
	tsMaxQuery := []bson.M{
		bson.M{"$project": bson.M{"ts_list": 1}}, // include the "ts_list" field from every document
		bson.M{"$unwind": "$ts_list"},            // concat all the lists together
		bson.M{"$sort": bson.M{"ts_list": -1}},   // sort all the timestamps from low to high
		bson.M{"$limit": 1},                      // take the lowest
	}

	// Execute query
	err2 := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.Structure.UniqueConnTable).Pipe(tsMaxQuery).One(&res)

	// Check for errors and parse results
	if err2 != nil {
		fmt.Println(err)
	} else {
		tsMax = res.Timestamp
	}

	return
}
