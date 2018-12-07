package structure

import (
	"github.com/activecm/rita/config"
	"github.com/activecm/rita/resources"

	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// GetConnSourcesFromDest finds all of the ips which communicated with a
// given destination ip
func GetConnSourcesFromDest(res *resources.Resources, ip string) []string {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	cons := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable)
	srcIter := cons.Find(bson.M{"dst": ip}).Iter()

	var srcStruct struct {
		Src string `bson:"src"`
	}
	var sources []string

	for srcIter.Next(&srcStruct) {
		sources = append(sources, srcStruct.Src)
	}
	return sources
}

//BuildUniqueConnectionsCollection finds the unique connection pairs
//between sources and destinations
func BuildUniqueConnectionsCollection(res *resources.Resources) {
	// Create the aggregate command
	sourceCollectionName,
		newCollectionName,
		newCollectionKeys,
		pipeline := getUniqueConnectionsScript(res.Config)

	err := res.DB.CreateCollection(newCollectionName, newCollectionKeys)
	if err != nil {
		res.Log.Error("Failed: ", newCollectionName, err.Error())
		return
	}

	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	res.DB.AggregateCollection(sourceCollectionName, ssn, pipeline)
}

func getUniqueConnectionsScript(conf *config.Config) (string, string, []mgo.Index, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	sourceCollectionName := conf.T.Structure.ConnTable

	// Name of the new collection
	newCollectionName := conf.T.Structure.UniqueConnTable

	// Desired Indexes
	keys := []mgo.Index{
		{Key: []string{"src", "dst"}, Unique: true},
		{Key: []string{"$hashed:src"}},
		{Key: []string{"$hashed:dst"}},
		{Key: []string{"connection_count"}},
	}

	// Aggregation to calculate various metrics (shown in the $project stage) that
	// occur between a unique IP pair. That is, all individual connections between two
	// given IPs will be summarized into a single entry in the resulting uconn collection.
	// We only process connections between IPs that cross the network border,
	// i.e. internal <-> external traffic.
	// This is mainly for performance (not having to process int<->int or ext<->ext)
	// but is also to reduce false positives as we are specifically looking for command
	// & control channels where a compromised internal system is communicating with an
	// attacker's server on the internet.
	pipeline := []bson.D{
		{
			// Only match on connections that are internal->external or external->internal
			// i.e. Exclude anything internal<->internal or external<->external
			{"$match", bson.M{
				"$or": []bson.M{
					bson.M{
						"$and": []bson.M{
							bson.M{"local_orig": true},
							bson.M{"local_resp": false},
						}},
					bson.M{
						"$and": []bson.M{
							bson.M{"local_orig": false},
							bson.M{"local_resp": true},
						}},
				}},
			},
		},
		{
			{"$group", bson.D{
				{"_id", bson.D{
					// In addition to defining the entry's key,
					// putting these here makes them available
					// for storing in the $project stage through $_id.*
					{"src", "$id_orig_h"},
					{"dst", "$id_resp_h"},
				}},
				// local_* is set per IP so we just need to know
				// any one of the connections' values
				{"ls", bson.D{{"$first", "$local_orig"}}},
				{"ld", bson.D{{"$first", "$local_resp"}}},
				// Total number of connections between two hosts
				{"conns", bson.D{
					{"$sum", 1},
				}},
				// Total number of bytes sent back and forth
				{"tbytes", bson.D{
					{"$sum", bson.D{
						{"$add", []interface{}{
							"$orig_ip_bytes",
							"$resp_ip_bytes",
						}},
					}},
				}},
				// Average number of bytes sent back and forth
				{"abytes", bson.D{
					{"$avg", bson.D{
						{"$add", []interface{}{
							"$orig_ip_bytes",
							"$resp_ip_bytes",
						}},
					}},
				}},
				// Array of all connection timestamps
				// $addToSet is used to ensure uniqueness of the values.
				// Duplicate values would result in the difference between
				// consecutive values being 0 in the beacon analysis and
				// would throw off the algorithm.
				{"ts", bson.D{{"$addToSet", "$ts"}}},
				// Array of bytes sent from origin in each connection
				// Here we want $push because every size is used as-is
				// instead of the difference of consecutive timestamps.
				{"orig_bytes", bson.D{{"$push", "$orig_ip_bytes"}}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"connection_count", "$conns"},
				{"src", "$_id.src"},
				{"dst", "$_id.dst"},
				{"local_src", "$ls"},
				{"local_dst", "$ld"},
				{"total_bytes", "$tbytes"},
				{"avg_bytes", "$abytes"},
				{"ts_list", "$ts"},
				{"orig_bytes_list", "$orig_bytes"},
			}},
		},
		{
			{"$out", newCollectionName},
		},
	}

	return sourceCollectionName, newCollectionName, keys, pipeline
}
