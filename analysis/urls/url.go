package urls

import (
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//BuildUrlsCollection performs url length analysis
func BuildUrlsCollection(res *database.Resources) {
	// Create the aggregate command
	sourceCollectionName,
		newCollectionName,
		newCollectionKeys,
		job,
		pipeline := getURLCollectionScript(res.System)

	// Create it
	err := res.DB.CreateCollection(newCollectionName, false, []mgo.Index{})
	if err != nil {
		res.Log.Error("Failed: ", newCollectionName, err.Error())
		return
	}

	// Map reduce it!
	if !res.DB.MapReduceCollection(sourceCollectionName, job) {
		return
	}

	ssn := res.DB.Session.Copy()
	defer ssn.Close()
	// Aggregate it
	res.DB.AggregateCollection(newCollectionName, ssn, pipeline)
	for _, index := range newCollectionKeys {
		ssn.DB(res.DB.GetSelectedDB()).C(res.System.UrlsConfig.UrlsTable).
			EnsureIndex(index)
	}
}

func getURLCollectionScript(sysCfg *config.SystemConfig) (string, string, []mgo.Index, mgo.MapReduce, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	sourceCollectionName := sysCfg.StructureConfig.HTTPTable

	// Name of the new collection
	newCollectionName := sysCfg.UrlsConfig.UrlsTable

	// Desired indeces
	keys := []mgo.Index{
		{Key: []string{"url", "uri"}, Unique: true},
		{Key: []string{"length"}},
	}

	// mgo passed MapReduce javascript function code
	job := mgo.MapReduce{
		Map: `function(){
					var result = {
						host: this.host,
						uri: this.uri,
						uid: this.uid,
						ip: this.id_resp_h,
						length: new NumberLong(this.host.length+this.uri.length)
					};
					emit(this._id, result);
				}`,
		Reduce: "function(key, values){return values}",
		Out:    bson.M{"replace": newCollectionName},
	}

	// nolint: vet
	pipeline := []bson.D{
		//this stage may be unneeded
		{
			{"$project", bson.D{
				{"_id", 0},
				{"url", "$value.host"},
				{"uri", "$value.uri"},
				{"ip", "$value.ip"},
				{"length", "$value.length"},
				{"uid", "$value.uid"},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", bson.D{
					{"url", "$url"},
					{"uri", "$uri"},
				}},
				{"ips", bson.D{
					{"$addToSet", "$ip"},
				}},
				{"length", bson.D{
					{"$first", "$length"},
				}},
				{"count", bson.D{
					{"$sum", 1},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"url", "$_id.url"},
				{"uri", "$_id.uri"},
				{"ips", 1},
				{"length", 1},
				{"count", 1},
			}},
		},
		{{
			"$out", newCollectionName,
		}},
	}

	return sourceCollectionName, newCollectionName, keys, job, pipeline
}
