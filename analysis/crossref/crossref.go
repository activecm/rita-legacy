package crossref

import (
	"sync"

	"github.com/ocmdev/rita/database"
)

type (
	XRefSelector interface {
		GetName() string                                           // the name of the analyis module
		Select(*database.Resources) (<-chan string, <-chan string) // returns the (internal, external) hosts
	}
)

func getXRefSelectors() []XRefSelector {
	beaconing := BeaconingSelector{}
	scanning := ScanningSelector{}

	return []XRefSelector{beaconing, scanning}
}

func BuildCrossrefCollection(res *database.Resources) {
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
	multiplexCrossref(res, "internXREF", internal)
	multiplexCrossref(res, "externXREF", external)
}

func multiplexCrossref(res *database.Resources, collection string,
	internalHosts map[string]<-chan string) {

	//each analysis type, while Mongo has awesome document level concurrency
	//writing the results from two different tests will lead to lock contention
	//however, we can spin up several threads per test type
	for name, hosts := range internalHosts {
		internWG := new(sync.WaitGroup)

		//create a number of threads to write
		//TODO config this value
		for i := 0; i < 2; i++ {
			internWG.Add(1)
			go writeCrossref(res, collection, name, hosts, internWG)
		}
		internWG.Wait()
	}
}

func writeCrossref(res *database.Resources, collection string, name string,
	hosts <-chan string, externWG *sync.WaitGroup) {

	for host := range hosts {
		//mongo upsert
		_ = host
	}
	externWG.Done()
}
