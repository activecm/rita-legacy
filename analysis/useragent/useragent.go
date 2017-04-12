package useragent

import (
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"

	"gopkg.in/mgo.v2/bson"
)

func BuildUserAgentCollection(res *database.Resources) {
	// Create the aggregate command
	source_collection_name,
		new_collection_name,
		new_collection_keys,
		pipeline := getUserAgentCollectionScript(res.System)

	// Create it
	error_check := res.DB.CreateCollection(new_collection_name, new_collection_keys)
	if error_check != "" {
		res.Log.Error("Failed: ", new_collection_name, error_check)
		return
	}

	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	// Aggregate it!
	res.DB.AggregateCollection(source_collection_name, ssn, pipeline)
}

func getUserAgentCollectionScript(sysCfg *config.SystemConfig) (string, string, []string, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	source_collection_name := sysCfg.StructureConfig.HttpTable

	// Name of the new collection
	new_collection_name := sysCfg.UserAgentConfig.UserAgentTable

	// Desired indeces
	keys := []string{"$hashed:user_agent"}

	// First aggregation script
	pipeline := []bson.D{
		{
			{"$group", bson.D{
				{"_id", "$user_agent"},
				{"uid", bson.D{
					{"$addToSet", "$uid"},
				}},
				{"times_used", bson.D{
					{"$sum", 1},
				}},
			}},
		},
		{
			{"$unwind", "$uid"},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"user_agent", "$_id"},
				{"uid", 1},
				{"times_used", 1},
			}},
		},
		{
			{"$out", new_collection_name},
		},
	}

	return source_collection_name, new_collection_name, keys, pipeline
}
