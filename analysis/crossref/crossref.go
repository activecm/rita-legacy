package crossref

import (
	"sync"

	"github.com/ocmdev/rita/database"
	dataXRef "github.com/ocmdev/rita/datatypes/crossref"
)

// getXRefSelectors is a place to add new selectors to the crossref module
func getXRefSelectors() []dataXRef.XRefSelector {
	beaconing := BeaconingSelector{}
	scanning := ScanningSelector{}

	return []dataXRef.XRefSelector{beaconing, scanning}
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

	//build internal and external at the same time
	//we could build the two collections at the same time
	//but, we have a thread for each analysis module reading,
	//this thread, and a number of write threads already spun.
	//TODO: config collection names
	multiplexXRef(res, res.System.CrossrefConfig.InternalTable, internal)
	multiplexXRef(res, res.System.CrossrefConfig.ExternalTable, external)
}

//multiplexXRef takes a target colllection, and a map from
//analysis module names to a channel containging the hosts associated with it
//and writes the incoming hosts to the target crossref collection
func multiplexXRef(res *database.Resources, collection string,
	analysisModules map[string]<-chan string) {

	xRefWG := new(sync.WaitGroup)
	for name, hosts := range analysisModules {
		xRefWG.Add(1)
		go writeXRef(res, collection, name, hosts, xRefWG)
	}
	xRefWG.Wait()
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
