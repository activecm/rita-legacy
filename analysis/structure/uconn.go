package structure

import (
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"

	"gopkg.in/mgo.v2/bson"
)

func BuildUniqueConnectionsCollection(res *database.Resources) {
	// Create the aggregate command
	source_collection_name,
		new_collection_name,
		new_collection_keys,
		pipeline := getUniqueConnectionsScript(res.System)

	// Aggregate it!
	error_check := res.DB.CreateCollection(new_collection_name, new_collection_keys)
	if error_check != "" {
		res.Log.Error("Failed: ", new_collection_name, error_check)
		return
	}

	// In case we need results
	results := []bson.M{}
	res.DB.AggregateCollection(source_collection_name, pipeline, &results)
}

func getUniqueConnectionsScript(sysCfg *config.SystemConfig) (string, string, []string, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	source_collection_name := sysCfg.StructureConfig.ConnTable

	// Name of the new collection
	new_collection_name := sysCfg.StructureConfig.UniqueConnTable

	// Desired Indeces
	keys := []string{"$hashed:src", "$hashed:dst"}

	// Aggregation script
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
				{"uid", 1},
			}},
		},
		{
			{"$out", new_collection_name},
		},
	}

	return source_collection_name, new_collection_name, keys, pipeline
}
