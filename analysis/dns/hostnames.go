package dns

import (
	"github.com/activecm/rita/config"
	dnsTypes "github.com/activecm/rita/datatypes/dns"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const tempHostnamesCollName string = "__temp_hostnames"

// BuildHostnamesCollection generates the mongo collection which maps
// hostnames to ip addresses
func BuildHostnamesCollection(res *resources.Resources) {
	sourceCollectionName,
		tempCollectionName,
		pipeline := getHostnamesAggregationScript(res.Config)

	hostNamesCollection := res.Config.T.DNS.HostnamesTable
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	res.DB.AggregateCollection(sourceCollectionName, ssn, pipeline)

	indexes := []mgo.Index{{Key: []string{"host"}, Unique: true}}
	err := res.DB.CreateCollection(hostNamesCollection, false, indexes)

	if err != nil {
		res.Log.Error("Could not create ", hostNamesCollection, err)
		return
	}

	mapHostnamesToIps(res.DB.GetSelectedDB(), tempCollectionName,
		hostNamesCollection, ssn)
	ssn.DB(res.DB.GetSelectedDB()).C(tempHostnamesCollName).DropCollection()
}

//getHostnamesAggregationScript maps dns a type queries to their answers
//unfortunately, answers may be other hostnames
func getHostnamesAggregationScript(conf *config.Config) (string, string, []bson.D) {
	sourceCollectionName := conf.T.Structure.DNSTable

	newCollectionName := tempHostnamesCollName

	// nolint: vet
	pipeline := []bson.D{
		{
			{"$match", bson.D{
				{"qtype_name", bson.D{
					{"$eq", "A"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"query", 1},
				{"answers", 1},
			}},
		},
		{
			{"$unwind", "$answers"},
		},
		{
			{"$group", bson.D{
				{"_id", "$query"},
				{"ips", bson.D{
					{"$addToSet", "$answers"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"host", "$_id"},
				{"ips", 1},
			}},
		},
		{
			{"$out", newCollectionName},
		},
	}
	return sourceCollectionName, newCollectionName, pipeline
}

//mapHostnamesToIps takes in a collection which maps dns A queries to answers
//and creates the hostname collection which maps hostnames to ip addresses
func mapHostnamesToIps(selectedDB string, sourceCollection string,
	destCollection string, session *mgo.Session) {
	dest := session.DB(selectedDB).C(destCollection)

	//run through the temp collection, determine which answers are
	//hostnames and which are ip addresses, and insert each hostname with
	//its associated ip adresses
	var mapping dnsTypes.Hostname
	iter := session.DB(selectedDB).C(sourceCollection).Find(nil).Iter()
	for iter.Next(&mapping) {
		hosts := []string{mapping.Host}
		var ips []string

		for _, answer := range mapping.IPs {
			if util.IsIP(answer) {
				ips = append(ips, answer)
			} else {
				hosts = append(hosts, answer)
			}
		}
		for _, host := range hosts {
			dest.Insert(dnsTypes.Hostname{Host: host, IPs: ips})
		}
	}
}

// GetIPsFromHost uses the hostnames table to do a cached whois query
func GetIPsFromHost(res *resources.Resources, host string) []string {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	hostnames := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.HostnamesTable)

	var destIPs dnsTypes.Hostname
	hostnames.Find(bson.M{"host": host}).One(&destIPs)

	return destIPs.IPs
}
