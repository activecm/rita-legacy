package crossref

import (
	"github.com/ocmdev/rita/analysis/blacklisted"
	"github.com/ocmdev/rita/analysis/urls"
	"github.com/ocmdev/rita/database"
	blacklistedData "github.com/ocmdev/rita/datatypes/blacklisted"
)

type (
	BlacklistedSelector struct{}
)

func (b BlacklistedSelector) GetName() string {
	return "blacklisted"
}

func (b BlacklistedSelector) Select(res *database.Resources) (<-chan string, <-chan string) {
	// make channels to return
	internalHosts := make(chan string)
	externalHosts := make(chan string)
	// run the read code async and return the channels immediately
	go func() {
		ssn := res.DB.Session.Copy()
		defer ssn.Close()

		iter := ssn.DB(res.DB.GetSelectedDB()).
			C(res.System.BlacklistedConfig.BlacklistTable).Find(nil).Iter()

		//iterate through blacklist table
		var data blacklistedData.Blacklist
		for iter.Next(&data) {

			//load the ips of those who visited the blacklisted site into the struct
			//and write them to xref
			blacklisted.SetBlacklistSources(res, &data)
			for _, src := range data.Sources {
				internalHosts <- src
			}

			//write the blacklisted site to xref, handle hostname appropriately
			if data.IsUrl {
				for _, dst := range urls.GetIPsFromHost(res, data.Host) {
					externalHosts <- dst
				}
			} else {
				externalHosts <- data.Host
			}
		}
		close(internalHosts)
		close(externalHosts)
	}()
	return internalHosts, externalHosts
}
