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

	// Aggregation script
	pipeline := []bson.D{
		{
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
					{"src", "$id_orig_h"},
					{"dst", "$id_resp_h"},
					{"ls", "$local_orig"},
					{"ld", "$local_resp"},
				}},
				{"conns", bson.D{
					{"$sum", 1},
				}},
				{"tbytes", bson.D{
					{"$sum", bson.D{
						{"$add", []interface{}{
							"$orig_ip_bytes",
							"$resp_ip_bytes",
						}},
					}},
				}},
				{"abytes", bson.D{
					{"$avg", bson.D{
						{"$add", []interface{}{
							"$orig_ip_bytes",
							"$resp_ip_bytes",
						}},
					}},
				}},
				{"tdur", bson.D{
					{"$sum", "$duration"},
				}},
				{"ts", bson.D{{"$push", "$ts"}}},
				{"orig_bytes", bson.D{{"$push", "$orig_ip_bytes"}}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"connection_count", "$conns"},
				{"src", "$_id.src"},
				{"dst", "$_id.dst"},
				{"local_src", "$_id.ls"},
				{"local_dst", "$_id.ld"},
				{"total_bytes", "$tbytes"},
				{"avg_bytes", "$abytes"},
				{"total_duration", "$tdur"},
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

// pipeline := []bson.D{
// 	{
// 		{"$group", bson.D{
// 			{"_id", bson.D{
// 				{"src", "$id_orig_h"},
// 				{"dst", "$id_resp_h"},
// 			}},
// 			{"connection_count", bson.D{
// 				{"$sum", 1},
// 			}},
// 			{"src", bson.D{
// 				{"$first", "$id_orig_h"},
// 			}},
// 			{"dst", bson.D{
// 				{"$first", "$id_resp_h"},
// 			}},
// 			{"local_src", bson.D{
// 				{"$first", "$local_orig"},
// 			}},
// 			{"local_dst", bson.D{
// 				{"$first", "$local_resp"},
// 			}},
// 			{"total_bytes", bson.D{
// 				{"$sum", bson.D{
// 					{"$add", []interface{}{
// 						"$orig_ip_bytes",
// 						"$resp_ip_bytes",
// 					}},
// 				}},
// 			}},
// 			{"avg_bytes", bson.D{
// 				{"$avg", bson.D{
// 					{"$add", []interface{}{
// 						"$orig_ip_bytes",
// 						"$resp_ip_bytes",
// 					}},
// 				}},
// 			}},
// 			{"total_duration", bson.D{
// 				{"$sum", "$duration"},
// 			}},
// 		}},
// 	},
// 	{
// 		{"$project", bson.D{
// 			{"_id", 0},
// 			{"connection_count", 1},
// 			{"src", 1},
// 			{"dst", 1},
// 			{"local_src", 1},
// 			{"local_dst", 1},
// 			{"total_bytes", 1},
// 			{"avg_bytes", 1},
// 			{"total_duration", 1},
// 		}},
// 	},
// 	{
// 		{"$out", newCollectionName},
// 	},
// }
