package scanning

import (
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"

	"gopkg.in/mgo.v2/bson"
)

func BuildScanningCollection(res *database.Resources) {
	// Create the aggregate command
	source_collection_name,
		new_collection_name,
		new_collection_keys,
		pipeline := getScanningCollectionScript(res.System)

	// Create it
	err := res.DB.CreateCollection(new_collection_name, new_collection_keys)
	if err != nil {
		res.Log.Error("Failed: ", new_collection_name, err.Error())
		return
	}
	ssn := res.DB.Session.Copy()
	defer ssn.Close()
	// Aggregate it!
	res.DB.AggregateCollection(source_collection_name, ssn, pipeline)
}

func getScanningCollectionScript(sysCfg *config.SystemConfig) (string, string, []string, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	source_collection_name := sysCfg.StructureConfig.ConnTable

	// Name of the new collection
	new_collection_name := sysCfg.ScanningConfig.ScanTable

	// Get scan threshold
	scan_thresh := sysCfg.ScanningConfig.ScanThreshold

	// Desired indeces
	keys := []string{"-port_count", "$hashed:src", "$hashed:dst"}

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
				{"port_set", bson.D{
					{"$addToSet", "$id_resp_p"},
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
				{"port_set", 1},
				{"port_count", bson.D{
					{"$size", "$port_set"},
				}},
			}},
		},
		{
			{"$match", bson.D{
				{"port_count", bson.D{
					{"$gt", scan_thresh},
				}},
			}},
		},
		{
			{"$sort", bson.D{
				{"port_count", -1},
			}},
		},
		{
			{"$out", new_collection_name},
		},
	}

	return source_collection_name, new_collection_name, keys, pipeline
}
