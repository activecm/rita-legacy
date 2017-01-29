package crossref

import (
	"sync"

	"github.com/ocmdev/rita/database"
)

type (
	//XRefSelector selects internal and external hosts from analysis modules
	XRefSelector interface {
		// GetName returns the name of the analyis module
		GetName() string
		// Select returns channels containgin the internal and external hosts
		Select(*database.Resources) (<-chan string, <-chan string)
	}
)

// getXRefSelectors is a place to add new selectors to the crossref module
func getXRefSelectors() []XRefSelector {
	beaconing := BeaconingSelector{}
	scanning := ScanningSelector{}

	return []XRefSelector{beaconing, scanning}
}

// BuildCrossrefCollection runs threaded crossref analysis
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
	//we could build the two collections at the same time
	//but, we have a thread for each analysis module reading,
	//this thread, and a number of write threads already spun.
	//TODO: config collection names
	multiplexCrossref(res, "internXREF", internal)
	multiplexCrossref(res, "externXREF", external)
}

//multiplexCrossref takes a target colllection, and a map from
//analysis module names to a channel containging the hosts associated with it
//and writes the incoming hosts to the target crossref collection
func multiplexCrossref(res *database.Resources, collection string,
	analysisModules map[string]<-chan string) {

	//While Mongo has awesome document level concurrency
	//writing the results from two different analysis modules will lead
	//to lock contention we can spin up several threads per test type
	for name, hosts := range analysisModules {
		internWG := new(sync.WaitGroup)

		//create a number of threads to write
		//TODO config this value
		for i := 0; i < 2; i++ {
			internWG.Add(1)
			go writeCrossref(res, collection, name, hosts, internWG)
		}
		internWG.Wait() //waitfor all of the writes for this analysis module
	}
}

// writeCrossref upserts a value into the target crossref collection
func writeCrossref(res *database.Resources, collection string, name string,
	hosts <-chan string, externWG *sync.WaitGroup) {

	for host := range hosts {
		//mongo upsert
		_ = host
	}
	externWG.Done()
}
