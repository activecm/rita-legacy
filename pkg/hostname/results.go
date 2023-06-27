package hostname

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
)

// HostnameIPResults returns the IP addresses the hostname was seen resolving to in the dataset
func HostnameIPResults(res *resources.Resources, hostname string) ([]data.UniqueIP, error) {
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
	}

	var ipResults []data.UniqueIP
	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.HostnamesTable).Pipe(ipsForHostnameQuery).AllowDiskUse().All(&ipResults)
	return ipResults, err
}