package crossref

import (
	"github.com/activecm/rita/analysis/beacon"
	"github.com/activecm/rita/database"
	dataBeacon "github.com/activecm/rita/datatypes/beacon"
)

type (
	//BeaconingSelector implements the XRefSelector interface for beaconing
	BeaconingSelector struct{}
)

//GetName returns "beaconing"
func (s BeaconingSelector) GetName() string {
	return "beaconing"
}

//Select selects beaconing hosts for XRef analysis
func (s BeaconingSelector) Select(res *database.Resources) (<-chan string, <-chan string) {
	// make channels to return
	sourceHosts := make(chan string)
	destHosts := make(chan string)
	// run the read code async and return the channels immediately
	go func() {
		ssn := res.DB.Session.Copy()
		defer ssn.Close()
		iter := beacon.GetBeaconResultsView(res, ssn, res.Config.S.Crossref.BeaconThreshold)

		//this will produce duplicates if multiple sources beaconed to the same dest
		//however, this is accounted for in the finalizing step of xref
		var data dataBeacon.BeaconAnalysisView
		for iter.Next(&data) {
			sourceHosts <- data.Src
			destHosts <- data.Dst
		}
		close(sourceHosts)
		close(destHosts)
	}()
	return sourceHosts, destHosts
}
