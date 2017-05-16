package useragent

import (
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"

	"gopkg.in/mgo.v2/bson"
)

//BuildUserAgentCollection performs frequency analysis on user agents
func BuildUserAgentCollection(res *database.Resources) {
	// Create the aggregate command
	sourceCollectionName,
		newCollectionName,
		newCollectionKeys,
		pipeline := getUserAgentCollectionScript(res.System)

	// Create it
	err := res.DB.CreateCollection(new_collection_name, new_collection_keys)
	if err != nil {
		res.Log.Error("Failed: ", new_collection_name, err.Error())
		return
	}

	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	// Aggregate it!
	res.DB.AggregateCollection(sourceCollectionName, ssn, pipeline)
}

func getUserAgentCollectionScript(sysCfg *config.SystemConfig) (string, string, []string, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	sourceCollectionName := sysCfg.StructureConfig.HTTPTable

	// Name of the new collection
	newCollectionName := sysCfg.UserAgentConfig.UserAgentTable

	// Desired indeces
	keys := []string{"-times_used"}

	// First aggregation script
	// nolint: vet
	pipeline := []bson.D{
		{
			{"$group", bson.D{
				{"_id", "$user_agent"},
				{"times_used", bson.D{
					{"$sum", 1},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"user_agent", "$_id"},
				{"times_used", 1},
			}},
		},
		{
			{"$out", newCollectionName},
		},
	}

	return sourceCollectionName, newCollectionName, keys, pipeline
}
