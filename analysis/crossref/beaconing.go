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
	internalHosts := make(chan string, 100)
	externalHosts := make(chan string, 100)
	// run the read code async and return the channels immediately
	go func() {
		iter := TBD.GetTBDResultsView(res, res.System.CrossrefConfig.TBDThreshold)

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
