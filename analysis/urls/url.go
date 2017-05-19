package urls

import (
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func BuildUrlsCollection(res *database.Resources) {
	// Create the aggregate command
	source_collection_name,
		new_collection_name,
		new_collection_keys,
		job,
		pipeline := getUrlCollectionScript(res.System)

	// Create it
	error_check := res.DB.CreateCollection(new_collection_name, new_collection_keys)
	if error_check != "" {
		res.Log.Error("Failed: ", new_collection_name, error_check)
		return
	}

	// Map reduce it!
	if !res.DB.MapReduceCollection(source_collection_name, job) {
		return
	}

	ssn := res.DB.Session.Copy()
	defer ssn.Close()
	// Aggregate it
	res.DB.AggregateCollection(new_collection_name, ssn, pipeline)
}

func getUrlCollectionScript(sysCfg *config.SystemConfig) (string, string, []string, mgo.MapReduce, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	source_collection_name := sysCfg.StructureConfig.HTTPTable

	// Name of the new collection
	new_collection_name := sysCfg.UrlsConfig.UrlsTable

	// Desired indeces
	keys := []string{"$hashed:url", "-length"}
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
		Out:    bson.M{"replace": new_collection_name},
	}

	// nolint: vet
	pipeline := []bson.D{
		{
			{"$project", bson.D{
				{"_id", 1},
				{"url", "$value.host"},
				{"uri", "$value.uri"},
				{"ip", "$value.ip"},
				{"length", "$value.length"},
				{"uid", "$value.uid"},
			}},
		},
		{
			{"$out", new_collection_name},
		},
	}

	return source_collection_name, new_collection_name, keys, job, pipeline
}
