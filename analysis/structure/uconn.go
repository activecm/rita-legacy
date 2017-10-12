package structure

import (
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// GetConnSourcesFromDest finds all of the ips which communicated with a
// given destination ip
func GetConnSourcesFromDest(res *database.Resources, ip string) []string {
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
func BuildUniqueConnectionsCollection(res *database.Resources) {
	// Create the aggregate command
	sourceCollectionName,
		newCollectionName,
		newCollectionKeys,
		pipeline := getUniqueConnectionsScript(res.Config)

	err := res.DB.CreateCollection(newCollectionName, true, newCollectionKeys)
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

	// Desired Indeces
	keys := []mgo.Index{
		{Key: []string{"src", "dst"}, Unique: true},
		{Key: []string{"$hashed:src"}},
		{Key: []string{"$hashed:dst"}},
	}

	// Aggregation script
	// nolint: vet
	pipeline := []bson.D{
		{
			{"$group", bson.D{
				{"_id", bson.D{
					{"src", "$id_origin_h"},
					{"dst", "$id_resp_h"},
				}},
				{"connection_count", bson.D{
					{"$sum", 1},
				}},
				{"src", bson.D{
					{"$first", "$id_origin_h"},
				}},
				{"dst", bson.D{
					{"$first", "$id_resp_h"},
				}},
				{"local_src", bson.D{
					{"$first", "$local_orig"},
				}},
				{"local_dst", bson.D{
					{"$first", "$local_resp"},
				}},
				{"total_bytes", bson.D{
					{"$sum", bson.D{
						{"$add", []interface{}{
							"$orig_ip_bytes",
							"$resp_ip_bytes",
						}},
					}},
				}},
				{"avg_bytes", bson.D{
					{"$avg", bson.D{
						{"$add", []interface{}{
							"$orig_ip_bytes",
							"$resp_ip_bytes",
						}},
					}},
				}},
				{"total_duration", bson.D{
					{"$sum", "$duration"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"connection_count", 1},
				{"src", 1},
				{"dst", 1},
				{"local_src", 1},
				{"local_dst", 1},
				{"total_bytes", 1},
				{"avg_bytes", 1},
				{"total_duration", 1},
			}},
		},
		{
			{"$out", newCollectionName},
		},
	}

	return sourceCollectionName, newCollectionName, keys, pipeline
}
