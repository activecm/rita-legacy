package dns

import (
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
	"gopkg.in/mgo.v2/bson"
)

// BuildHostnamesCollection generates the mongo collection which maps
// hostnames to ip addresses
func BuildHostnamesCollection(res *database.Resources) {
	sourceCollectionName,
		newCollectionName,
		newCollectionKeys,
		pipeline := getHostnamesAggregationScript(res.System)

	err := res.DB.CreateCollection(newCollectionName, newCollectionKeys)
	if err != "" {
		res.Log.Error("Failed: ", newCollectionName, err)
		return
	}

	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	res.DB.AggregateCollection(sourceCollectionName, ssn, pipeline)
}

func getHostnamesAggregationScript(sysCfg *config.SystemConfig) (string, string, []string, []bson.D) {
	sourceCollectionName := sysCfg.StructureConfig.DnsTable

	newCollectionName := sysCfg.UrlsConfig.HostnamesTable

	keys := []string{"$hashed:host"}

	pipeline := []bson.D{
		{
			{"$match", bson.D{
				{"qtype_name", bson.D{
					{"$eq", "A"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"query", 1},
				{"answers", 1},
			}},
		},
		{
			{"$unwind", "$answers"},
		},
		{
			{"$group", bson.D{
				{"_id", "$query"},
				{"ips", bson.D{
					{"$addToSet", "$answers"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"host", "$_id"},
				{"ips", 1},
			}},
		},
		{
			{"$out", newCollectionName},
		},
	}
	return sourceCollectionName, newCollectionName, keys, pipeline
}
