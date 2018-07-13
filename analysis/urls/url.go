package urls

import (
	"github.com/activecm/rita/config"
	"github.com/activecm/rita/resources"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

//BuildUrlsCollection performs url length analysis
func BuildUrlsCollection(res *resources.Resources) {
	// Create the aggregate command
	sourceCollectionName,
		newCollectionName,
		newCollectionKeys,
		job,
		pipeline := getURLCollectionScript(res.Config)

	// Create it
	err := res.DB.CreateCollection(newCollectionName, []mgo.Index{})
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
		ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Urls.UrlsTable).
			EnsureIndex(index)
	}
}

func getURLCollectionScript(conf *config.Config) (string, string, []mgo.Index, mgo.MapReduce, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	sourceCollectionName := conf.T.Structure.HTTPTable

	// Name of the new collection
	newCollectionName := conf.T.Urls.UrlsTable

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
