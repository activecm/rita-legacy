package hostname

import (
	"github.com/activecm/rita-legacy/pkg/data"
	"github.com/activecm/rita-legacy/resources"
	"github.com/globalsign/mgo/bson"
)

// IPResults returns the IP addresses the hostname was seen resolving to in the dataset
func IPResults(res *resources.Resources, hostname string) ([]data.UniqueIP, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	ipsForHostnameQuery := []bson.M{
		{"$match": bson.M{
			"host": hostname,
		}},
		{"$project": bson.M{
			"ips": "$dat.ips",
		}},
		{"$unwind": "$ips"},
		{"$unwind": "$ips"},
		{"$group": bson.M{
			"_id": bson.M{
				"ip":           "$ips.ip",
				"network_uuid": "$ips.network_uuid",
			},
			"network_name": bson.M{"$last": "$ips.network_name"},
		}},
		{"$project": bson.M{
			"_id":          0,
			"ip":           "$_id.ip",
			"network_uuid": "$_id.network_uuid",
			"network_name": "$network_name",
		}},
		{"$sort": bson.M{
			"ip": 1,
		}},
	}

	var ipResults []data.UniqueIP
	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.HostnamesTable).Pipe(ipsForHostnameQuery).AllowDiskUse().All(&ipResults)
	return ipResults, err
}

// FQDNResults returns the FQDNs the IP address was seen resolving to in the dataset
func FQDNResults(res *resources.Resources, hostIP string) ([]*FQDNResult, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	fqdnsForHostnameQuery := []bson.M{
		{"$match": bson.M{
			"dat.ips.ip": hostIP,
		}},
		{"$group": bson.M{
			"_id": "$host",
		}},
	}

	var fqdnResults []*FQDNResult
	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.HostnamesTable).Pipe(fqdnsForHostnameQuery).AllowDiskUse().All(&fqdnResults)
	return fqdnResults, err
}
