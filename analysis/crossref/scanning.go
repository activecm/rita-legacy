package crossref

import (
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/scanning"
)

type (
	ScanningSelector struct{}
)

func (s ScanningSelector) GetName() string {
	return "scanning"
}

func (s ScanningSelector) Select(res *database.Resources) (<-chan string, <-chan string) {
	// make channels to return
	internalHosts := make(chan string)
	externalHosts := make(chan string)
	// run the read code async and return the channels immediately
	go func() {
		ssn := res.DB.Session.Copy()
		defer ssn.Close()
		iter := ssn.DB(res.DB.GetSelectedDB()).
			C(res.System.ScanningConfig.ScanTable).Find(nil).Iter()

		var data scanning.Scan
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
