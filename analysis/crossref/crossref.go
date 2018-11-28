package crossref

import (
	"sync"

	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	dataXRef "github.com/activecm/rita/datatypes/crossref"
	"github.com/activecm/rita/resources"
)

// getXRefSelectors is a place to add new selectors to the crossref module
func getXRefSelectors() []dataXRef.XRefSelector {
	beaconing := BeaconingSelector{}
	blSourceIPs := BLSourceIPSelector{}
	blDestIPs := BLDestIPSelector{}

	return []dataXRef.XRefSelector{beaconing, blSourceIPs, blDestIPs}
}

// BuildXRefCollection runs threaded crossref analysis
func BuildXRefCollection(res *resources.Resources) {
	indexes := []mgo.Index{{Key: []string{"host"}, Unique: true}}
	res.DB.CreateCollection(res.Config.T.Crossref.SourceTable, indexes)
	res.DB.CreateCollection(res.Config.T.Crossref.DestTable, indexes)

	//maps from analysis types to channels of hosts found
	sources := make(map[string]<-chan string)
	destinations := make(map[string]<-chan string)

	//kick off reads
	for _, selector := range getXRefSelectors() {
		sourceHosts, destinationHosts := selector.Select(res)
		sources[selector.GetName()] = sourceHosts
		destinations[selector.GetName()] = destinationHosts
	}

	xRefWG := new(sync.WaitGroup)
	xRefWG.Add(2)
	//kick off writes
	go multiplexXRef(res, res.Config.T.Crossref.SourceTable, sources, xRefWG)
	go multiplexXRef(res, res.Config.T.Crossref.DestTable, destinations, xRefWG)
	xRefWG.Wait()

	//group by host ip and put module findings into an array
	finalizeXRef(res, res.Config.T.Crossref.SourceTable)
	finalizeXRef(res, res.Config.T.Crossref.DestTable)
}

//multiplexXRef takes a target colllection, and a map from
//analysis module names to a channel containging the hosts associated with it
//and writes the incoming hosts to the target crossref collection
func multiplexXRef(res *resources.Resources, collection string,
	analysisModules map[string]<-chan string, externWG *sync.WaitGroup) {

	xRefWG := new(sync.WaitGroup)
	for name, hosts := range analysisModules {
		xRefWG.Add(1)
		go writeXRef(res, collection, name, hosts, xRefWG)
	}
	xRefWG.Wait()
	externWG.Done()
}

// writeXRef upserts a value into the target crossref collection
func writeXRef(res *resources.Resources, collection string,
	moduleName string, hosts <-chan string, externWG *sync.WaitGroup) {

	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	for host := range hosts {
		data := dataXRef.XRef{
			ModuleName: moduleName,
			Host:       host,
		}
		ssn.DB(res.DB.GetSelectedDB()).C(collection).Insert(data)
	}
	externWG.Done()
}

func finalizeXRef(res *resources.Resources, collection string) {
	// Aggregation script
	// nolint: vet
	pipeline := []bson.D{
		{
			{"$group", bson.D{
				{"_id", bson.D{
					{"host", "$host"},
				}},
				{"modules", bson.D{
					{"$addToSet", "$module"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"host", "$_id.host"},
				{"modules", 1},
			}},
		},
		{
			{"$out", collection},
		},
	}
	ssn := res.DB.Session.Copy()
	res.DB.AggregateCollection(collection, ssn, pipeline)
	ssn.Close()
}
