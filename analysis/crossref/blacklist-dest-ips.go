package crossref

import (
	"github.com/activecm/rita/datatypes/blacklist"
	"github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
)

type (
	//BLDestIPSelector implements the XRefSelector interface for blacklisted destination ips
	BLDestIPSelector struct{}
)

//GetName returns "bl-dest-ips"
func (s BLDestIPSelector) GetName() string {
	return "bl-dest-ip"
}

//Select selects blacklisted dest ips for XRef analysis
func (s BLDestIPSelector) Select(res *resources.Resources) (<-chan string, <-chan string) {
	// make channels to return
	sourceHosts := make(chan string)
	destHosts := make(chan string)
	// run the read code async and return the channels immediately
	go func() {
		ssn := res.DB.Session.Copy()
		defer ssn.Close()

		var blIPs []blacklist.BlacklistedIP
		ssn.DB(res.DB.GetSelectedDB()).
			C(res.Config.T.Blacklisted.DestIPsTable).
			Find(nil).All(&blIPs)

		for _, ip := range blIPs {
			var connected []structure.UniqueConnection
			ssn.DB(res.DB.GetSelectedDB()).
				C(res.Config.T.Structure.UniqueConnTable).Find(
				bson.M{"dst": ip.IP},
			).All(&connected)
			for _, uconn := range connected {
				sourceHosts <- uconn.Src
			}
			destHosts <- ip.IP
		}
		close(sourceHosts)
		close(destHosts)
	}()
	return sourceHosts, destHosts
}
