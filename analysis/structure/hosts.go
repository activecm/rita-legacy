package structure

import (
	"github.com/ocmdev/rita/config"
	"gopkg.in/mgo.v2/bson"
)

func GetHosts(sysCfg *config.SystemConfig) (string, string, []string, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	source_collection_name := sysCfg.StructureConfig.ConnTable

	// Name of the new collection
	new_collection_name := sysCfg.StructureConfig.HostTable

	// Desired indeces
	keys := []string{"$hashed:ip", "local"}

	// Aggregation script
	pipeline := []bson.D{
		{
			{"$project", bson.D{
				{"hosts", []interface{}{
					bson.D{
						{"ip", "$id_origin_h"},
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
			}},
		},
		{
			{"$out", new_collection_name},
		},
	}

	return source_collection_name, new_collection_name, keys, pipeline
}
