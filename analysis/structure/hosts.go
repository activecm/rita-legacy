package structure

import (
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
	"gopkg.in/mgo.v2/bson"
)

// BuildHostsCollection builds the 'host' collection for this timeframe. Note
// that this is a different host collection that the one found in HostsIntelDB.
// This host collection references only hosts found in this time frame, info
// from the HostsIntelDB collection can be found by following the 'intelid' field
// after it is populated by the cymru and blacklist modules. Runs via mongodb
// aggregation. Sourced from the 'conn' table.
// TODO: Confirm that this section of code is not faster than an aggregation from
// the 'uconn' table which should have less repeated data.
func BuildHostsCollection(res *database.Resources) {
	// Create the aggregate command
	source_collection_name,
		new_collection_name,
		new_collection_keys,
		pipeline := getHosts(res.System)

	// Aggregate it!
	error_check := res.DB.CreateCollection(new_collection_name, new_collection_keys)
	if error_check != "" {
		res.Log.Error("Failed: ", new_collection_name, error_check)
		return
	}

	res.DB.AggregateCollection(source_collection_name, pipeline)
}

func getHosts(sysCfg *config.SystemConfig) (string, string, []string, []bson.D) {
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
