package crossref

import (
	"github.com/bglebrun/rita/analysis/beacon"
	"github.com/bglebrun/rita/database"
	dataBeacon "github.com/bglebrun/rita/datatypes/beacon"
)

type (
	BeaconingSelector struct{}
)

func (s BeaconingSelector) GetName() string {
	return "beaconing"
}

func (s BeaconingSelector) Select(res *database.Resources) (<-chan string, <-chan string) {
	// make channels to return
	internalHosts := make(chan string)
	externalHosts := make(chan string)
	// run the read code async and return the channels immediately
	go func() {
		ssn := res.DB.Session.Copy()
		defer ssn.Close()
		iter := beacon.GetBeaconResultsView(res, ssn, res.System.CrossrefConfig.BeaconThreshold)

		//this will produce duplicates if multiple sources beaconed to the same dest
		//however, this is accounted for in the finalizing step of xref
		var data dataBeacon.BeaconAnalysisView
		for iter.Next(&data) {
			if data.LocalSrc {
				internalHosts <- data.Src
			} else {
				externalHosts <- data.Src
			}
			if data.LocalDst {
				internalHosts <- data.Dst
			} else {
				externalHosts <- data.Dst
			}
		}
		close(internalHosts)
		close(externalHosts)
	}()
	return internalHosts, externalHosts
}
