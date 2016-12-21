package urls

import (
	"github.com/ocmdev/rita/config"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func GetUrlCollectionScript(sysCfg *config.SystemConfig) (string, string, []string, mgo.MapReduce, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	source_collection_name := sysCfg.StructureConfig.HttpTable

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

func GetHostnamesAggregationScript(sysCfg *config.SystemConfig) (string, string, []string, []bson.D) {
	source_collection_name := sysCfg.UrlsConfig.UrlsTable

	new_collection_name := sysCfg.UrlsConfig.HostnamesTable

	keys := []string{"$hashed:host"}

	pipeline := []bson.D{
		{
			{"$project", bson.D{
				{"_id", 0},
				{"url", 1},
				{"ip", 1},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$url"},
				{"ips", bson.D{
					{"$addToSet", "$ip"},
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
			{"$out", new_collection_name},
		},
	}
	return source_collection_name, new_collection_name, keys, pipeline
}
