package crossref

import (
	"github.com/ocmdev/rita/analysis/TBD"
	"github.com/ocmdev/rita/database"
	dataTBD "github.com/ocmdev/rita/datatypes/TBD"
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
		iter := TBD.GetTBDResultsView(res, ssn, res.System.CrossrefConfig.TBDThreshold)

		var data dataTBD.TBDAnalysisView
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
