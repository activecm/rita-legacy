package useragent

import (
	"github.com/bglebrun/rita/config"

	"gopkg.in/mgo.v2/bson"
)

func GetUserAgentCollectionScript(sysCfg *config.SystemConfig) (string, string, []string, []bson.D) {
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
