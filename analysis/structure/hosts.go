package structure

import (
	"github.com/activecm/rita/config"
	"github.com/activecm/rita/resources"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// BuildHostsCollection builds the 'host' collection for this timeframe.
// Runs via mongodb aggregation. Sourced from the 'conn' table.
func BuildHostsCollection(res *resources.Resources) {
	// Create the aggregate command
	sourceCollectionName,
		newCollectionName,
		newCollectionKeys,
		pipeline := getHosts(res.Config)

	// Aggregate it!
	errorCheck := res.DB.CreateCollection(newCollectionName, false, newCollectionKeys)
	if errorCheck != nil {
		res.Log.Error("Failed: ", newCollectionName, errorCheck)
		return
	}

	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	res.DB.AggregateCollection(sourceCollectionName, ssn, pipeline)
}

//getHosts aggregates the individual hosts from the conn collection and
//labels them as private or public as well as ipv4 or ipv6. The aggregation
//includes padding for a binary encoding of the ip address.
func getHosts(conf *config.Config) (string, string, []mgo.Index, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	sourceCollectionName := conf.T.Structure.ConnTable

	// Name of the new collection
	newCollectionName := conf.T.Structure.HostTable

	// Desired indeces
	keys := []mgo.Index{
		{Key: []string{"ip"}, Unique: true},
		{Key: []string{"local"}},
		{Key: []string{"ipv4"}},
	}

	// Aggregation script
	// nolint: vet
	pipeline := []bson.D{
		{
			{"$project", bson.D{
				{"hosts", []interface{}{
					bson.D{
						{"ip", "$id_orig_h"},
						{"local", "$local_orig"},
					},
					bson.D{
						{"ip", "$id_resp_h"},
						{"local", "$local_resp"},
					},
				}},
			}},
		},
		{
			{"$unwind", "$hosts"},
		},
		{
			{"$group", bson.D{
				{"_id", "$hosts.ip"},
				{"local", bson.D{
					{"$first", "$hosts.local"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"ip", "$_id"},
				{"local", 1},
				{"ipv4", bson.D{
					{"$cond", bson.D{
						{"if", bson.D{
							{"$eq", []interface{}{
								bson.D{
									{"$indexOfCP", []interface{}{
										"$_id", ":",
									}},
								},
								-1,
							}},
						}},
						{"then", bson.D{
							{"$literal", true},
						}},
						{"else", bson.D{
							{"$literal", false},
						}},
					}},
				}},
			}},
		},
		{
			{"$out", newCollectionName},
		},
	}

	return sourceCollectionName, newCollectionName, keys, pipeline
}
