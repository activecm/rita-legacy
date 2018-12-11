package blacklist

import (
	"runtime"

	"github.com/activecm/rita/datatypes/blacklist"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

func getUniqueIPFromUconnPipeline(field string) []bson.D {
	//nolint: vet
	return []bson.D{
		{
			{"$group", bson.M{
				"_id":         "$" + field,
				"total_bytes": bson.M{"$sum": "$total_bytes"},
				"avg_bytes":   bson.M{"$sum": "$avg_bytes"},
				"conn_count":  bson.M{"$sum": "$connection_count"},
				"uconn_count": bson.M{"$sum": 1},
				"targets":     bson.M{"$push": "$src"},
			}},
		},
		{
			{"$project", bson.M{
				"_id":         0,
				"ip":          "$_id",
				"total_bytes": 1,
				"avg_bytes":   1,
				"conn_count":  1,
				"uconn_count": 1,
				"targets":     1,
			}},
		},
	}
}

//buildBlacklistedIPs builds a set of blacklisted ips from the
//iterator provided, the system config, a handle to rita-blacklist,
//a buffer of ips to check at a time, and a boolean designating
//whether or not the ips are connection sources or destinations
func buildBlacklistedIPs(res *resources.Resources, source bool) {
	//create session to write to
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	//choose the output collection and assign iterator
	var collectionName string
	var ipIter *mgo.Iter
	if source {
		collectionName = res.Config.T.Blacklisted.SourceIPsTable
		uniqueSourcesAggregation := getUniqueIPFromUconnPipeline("src")
		ipIter = res.DB.AggregateCollection(
			res.Config.T.Structure.UniqueConnTable,
			ssn,
			uniqueSourcesAggregation,
		)
	} else {
		collectionName = res.Config.T.Blacklisted.DestIPsTable
		uniqueDestAggregation := getUniqueIPFromUconnPipeline("dst")
		ipIter = res.DB.AggregateCollection(
			res.Config.T.Structure.UniqueConnTable,
			ssn,
			uniqueDestAggregation,
		)
	}

	// create the actual collection
	collectionKeys := []mgo.Index{
		{Key: []string{"$hashed:ip"}},
	}
	err := res.DB.CreateCollection(collectionName, collectionKeys)
	if err != nil {
		res.Log.Error("Failed: ", collectionName, err.Error())
		return
	}

	// Lets run the analysis
	checkRitaBlacklistIPs(ipIter, res, source)

}

func checkRitaBlacklistIPs(ips *mgo.Iter, res *resources.Resources, source bool) {
	// fmt.Println("-- checkRitaBlacklist IPs --")
	count := 0

	//Create the workers
	writerWorker := newWriter(source, res.DB, res.Config)
	analyzerWorker := newAnalyzer(
		source,
		res.DB,
		res.Config,
		writerWorker.write,
		writerWorker.close,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		analyzerWorker.start()
		writerWorker.start()
	}

	var uconnRes struct {
		IP                string   `bson:"ip"`
		Connections       int      `bson:"conn_count"`
		UniqueConnections int      `bson:"uconn_count"`
		TotalBytes        int      `bson:"total_bytes"`
		AverageBytes      int      `bson:"avg_bytes"`
		Targets           []string `bson:"targets"`
	}

	for ips.Next(&uconnRes) {

		newInput := &blacklist.AnalysisInput{
			IP:                uconnRes.IP,
			Connections:       uconnRes.Connections,
			UniqueConnections: uconnRes.UniqueConnections,
			TotalBytes:        uconnRes.TotalBytes,
			AverageBytes:      uconnRes.AverageBytes,
			Targets:           uconnRes.Targets,
		}
		analyzerWorker.analyze(newInput)
		count++
	}

}
