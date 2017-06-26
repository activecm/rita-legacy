package crossref

import (
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/scanning"
)

type (
	//ScanningSelector implements the XRefSelector interface for scanning
	ScanningSelector struct{}
)

//GetName returns "scanning"
func (s ScanningSelector) GetName() string {
	return "scanning"
}

//Select selects scanning and scanned hosts for XRef analysis
func (s ScanningSelector) Select(res *database.Resources) (<-chan string, <-chan string) {
	// make channels to return
	sourceHosts := make(chan string)
	destHosts := make(chan string)
	// run the read code async and return the channels immediately
	go func() {
		ssn := res.DB.Session.Copy()
		defer ssn.Close()
		iter := ssn.DB(res.DB.GetSelectedDB()).
			C(res.System.ScanningConfig.ScanTable).Find(nil).Iter()

		var data scanning.Scan
		for iter.Next(&data) {
			sourceHosts <- data.Src
			destHosts <- data.Dst
		}
		close(sourceHosts)
		close(destHosts)
	}()
	return sourceHosts, destHosts
}
