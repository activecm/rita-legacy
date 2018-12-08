package structure

import (
	"github.com/activecm/rita/config"
	"github.com/activecm/rita/resources"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
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
	errorCheck := res.DB.CreateCollection(newCollectionName, newCollectionKeys)
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
	sourceCollectionName := conf.T.Structure.UniqueConnTable

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
						{"ip", "$src"},
						{"local", "$local_src"},
						{"src", true},
					},
					bson.D{
						{"ip", "$dst"},
						{"local", "$local_dst"},
						{"dst", true},
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
				{"src", bson.D{
					{"$push", "$hosts.src"},
				}},
				{"dst", bson.D{
					{"$push", "$hosts.dst"},
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
				{"count_src", bson.D{
					{"$size", "$src"},
				}},
				{"count_dst", bson.D{
					{"$size", "$dst"},
				}},
			}},
		},
		{
			{"$out", newCollectionName},
		},
	}

	return sourceCollectionName, newCollectionName, keys, pipeline
}

// troubleshooting:
//
// db.getCollection('uconn').aggregate([
//     {"$project":{
//             "hosts": [
//                     {
//                             "ip": "$src",
//                             "local": "$local_src",
//                             "src" : true,
//                     },
//                     {
//                             "ip": "$dst",
//                             "local": "$local_dst",
//                             "dst":true,
//                     },
//             ],
//     }},
//     {"$unwind": "$hosts"},
//     {"$group":{
//             "_id": "$hosts.ip",
//             "src":{$push:"$hosts.src"},
//             "dst":{$push:"$hosts.dst"},
// //             "count":{$sum:1},
//             "local": {"$first": "$hosts.local"},
//     }},
//     {"$project":{
//             "_id": 0,
//             "ip": "$_id",
//             "local": 1,
//             "count_src":{$size:"$src"},
//             "count_dst":{$size:"$dst"},
//             "ipv4": {$cond:{
//                             if:{"$eq": [{"$indexOfCP": ["$_id", ":"]},-1]},
//                             then: {"$literal":true},
//                             else:{"$literal": false},
//                         }},
//
//     }},
//     {$sort:{"count_src":-1}}
// ])
