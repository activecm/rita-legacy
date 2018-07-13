package crossref

import (
	"github.com/activecm/rita/datatypes/blacklist"
	"github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
)

type (
	//BLSourceIPSelector implements the XRefSelector interface for blacklisted source ips
	BLSourceIPSelector struct{}
)

//GetName returns "bl-source-ip"
func (s BLSourceIPSelector) GetName() string {
	return "bl-source-ip"
}

//Select selects blacklisted source ips for XRef analysis
func (s BLSourceIPSelector) Select(res *resources.Resources) (<-chan string, <-chan string) {
	// make channels to return
	sourceHosts := make(chan string)
	destHosts := make(chan string)
	// run the read code async and return the channels immediately
	go func() {
		ssn := res.DB.Session.Copy()
		defer ssn.Close()

		var blIPs []blacklist.BlacklistedIP
		ssn.DB(res.DB.GetSelectedDB()).
			C(res.Config.T.Blacklisted.SourceIPsTable).
			Find(nil).All(&blIPs)

		for _, ip := range blIPs {
			var connected []structure.UniqueConnection
			ssn.DB(res.DB.GetSelectedDB()).
				C(res.Config.T.Structure.UniqueConnTable).Find(
				bson.M{"src": ip.IP},
			).All(&connected)
			for _, uconn := range connected {
				destHosts <- uconn.Dst
			}
			sourceHosts <- ip.IP
		}
		close(sourceHosts)
		close(destHosts)
	}()
	return sourceHosts, destHosts
}
