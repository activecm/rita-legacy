package crossref

import (
	"sync"

	"gopkg.in/mgo.v2/bson"

	"github.com/ocmdev/rita/database"
	dataXRef "github.com/ocmdev/rita/datatypes/crossref"
)

// getXRefSelectors is a place to add new selectors to the crossref module
func getXRefSelectors() []dataXRef.XRefSelector {
	beaconing := BeaconingSelector{}
	scanning := ScanningSelector{}
	blacklisted := BlacklistedSelector{}

	return []dataXRef.XRefSelector{beaconing, scanning, blacklisted}
}

// BuildXRefCollection runs threaded crossref analysis
func BuildXRefCollection(res *database.Resources) {
	res.DB.CreateCollection(res.System.CrossrefConfig.InternalTable, []string{"host"})
	res.DB.CreateCollection(res.System.CrossrefConfig.ExternalTable, []string{"host"})

	//maps from analysis types to channels of hosts found
	internal := make(map[string]<-chan string)
	external := make(map[string]<-chan string)

	//kick off reads
	for _, selector := range getXRefSelectors() {
		internalHosts, externalHosts := selector.Select(res)
		internal[selector.GetName()] = internalHosts
		external[selector.GetName()] = externalHosts
	}

	xRefWG := new(sync.WaitGroup)
	xRefWG.Add(2)
	//kick off writes
	go multiplexXRef(res, res.System.CrossrefConfig.InternalTable, internal, xRefWG)
	go multiplexXRef(res, res.System.CrossrefConfig.ExternalTable, external, xRefWG)
	xRefWG.Wait()

	//group by host ip and put module findings into an array
	finalizeXRef(res, res.System.CrossrefConfig.InternalTable)
	finalizeXRef(res, res.System.CrossrefConfig.ExternalTable)
}

//multiplexXRef takes a target colllection, and a map from
//analysis module names to a channel containging the hosts associated with it
//and writes the incoming hosts to the target crossref collection
func multiplexXRef(res *database.Resources, collection string,
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
func writeXRef(res *database.Resources, collection string,
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

func finalizeXRef(res *database.Resources, collection string) {
	// Aggregation script
	// nolint: vet
	pipeline := []bson.D{
		{
			{"$group", bson.D{
				{"_id", bson.D{
					{"host", "$host"},
				}},
				{"host", bson.D{
					{"$first", "$host"},
				}},
				{"modules", bson.D{
					{"$addToSet", "$module"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"host", 1},
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
