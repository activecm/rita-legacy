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
	targetField := "dst"
	if field == "dst" {
		targetField = "src"
	}
	return []bson.D{
		{
			{"$group", bson.M{
				"_id":         "$" + field,
				"total_bytes": bson.M{"$sum": "$total_bytes"},
				"avg_bytes":   bson.M{"$sum": "$avg_bytes"},
				"conn_count":  bson.M{"$sum": "$connection_count"},
				"uconn_count": bson.M{"$sum": 1},
				"targets":     bson.M{"$push": "$" + targetField},
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

func getUniqueHostnameFromUconnPipeline(hosts []string) []bson.D {
	//nolint: vet
	return []bson.D{
		{
			{"$match", bson.M{
				"dst": bson.M{"$in": hosts},
			}},
		},
		{
			{"$group", bson.M{
				"_id":         "$dst",
				"total_bytes": bson.M{"$sum": "$total_bytes"},
				"avg_bytes":   bson.M{"$sum": "$avg_bytes"},
				"conn_count":  bson.M{"$sum": "$connection_count"},
				"uconn_count": bson.M{"$sum": 1},
				"targets":     bson.M{"$push": "$src"},
			}},
		},
		{
			{"$project", bson.M{
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
//the blacklist source or dest IPs collection, depending on the
//value of the source bool passed in.
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

	//run the analysis
	checkRitaBlacklistIPs(ipIter, res, source, collectionName)

}

//buildBlacklistedHostnames builds a set of blacklisted ips from the
func buildBlacklistedHostnames(res *resources.Resources) {
	//create session to write to
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	//choose the output collection and assign iterator
	var collectionName = res.Config.T.Blacklisted.HostnamesTable
	ipIter := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.HostnamesTable).Find(nil).Iter()

	// create the actual collection
	collectionKeys := []mgo.Index{
		{Key: []string{"$hashed:hostname"}},
	}
	err := res.DB.CreateCollection(collectionName, collectionKeys)
	if err != nil {
		res.Log.Error("Failed: ", collectionName, err.Error())
		return
	}

	// Lets run the analysis
	checkRitaBlacklistHostnames(ipIter, res, collectionName)

}

func checkRitaBlacklistIPs(ips *mgo.Iter, res *resources.Resources, source bool, collectionName string) {
	//Create the workers
	writerWorker := newWriter(collectionName, res.DB, res.Config)
	analyzerWorker := newIPAnalyzer(
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

		newInput := &blacklist.IPAnalysisInput{
			IP:                uconnRes.IP,
			Connections:       uconnRes.Connections,
			UniqueConnections: uconnRes.UniqueConnections,
			TotalBytes:        uconnRes.TotalBytes,
			AverageBytes:      uconnRes.AverageBytes,
			Targets:           uconnRes.Targets,
		}
		analyzerWorker.analyzeIP(newInput)
	}

}

func checkRitaBlacklistHostnames(ips *mgo.Iter, res *resources.Resources, collectionName string) {

	//Create the workers
	writerWorker := newWriter(collectionName, res.DB, res.Config)
	analyzerWorker := newHostnameAnalyzer(
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

	var hostnameRes struct {
		Host string   `bson:"host"`
		IPs  []string `bson:"ips"`
	}

	for ips.Next(&hostnameRes) {

		newInput := &blacklist.HostnameAnalysisInput{
			Host: hostnameRes.Host,
			IPs:  hostnameRes.IPs,
		}
		analyzerWorker.analyzeHostname(newInput)

	}

}
