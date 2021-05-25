package uconn

import (
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
)

//LongConnResults returns long connections longer than the given thresh in
//seconds. The results will be sorted, descending by duration.
//limit and noLimit control how many results are returned.
func LongConnResults(res *resources.Resources, thresh int, limit int, noLimit bool) ([]LongConnResult, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var longConnResults []LongConnResult

	longConnQuery := []bson.M{
		bson.M{"$match": bson.M{"dat.maxdur": bson.M{"$gt": thresh}}},
		bson.M{"$project": bson.M{
			"src":              1,
			"src_network_uuid": 1,
			"src_network_name": 1,
			"dst":              1,
			"dst_network_uuid": 1,
			"dst_network_name": 1,
			"maxdur":           "$dat.maxdur",
			"tuples":           bson.M{"$ifNull": []interface{}{"$dat.tuples", []interface{}{}}},
			// If entry for open_duration is missing or 0, then there are no open connections
			// from the src to the dst. If it is present and greater than 0, an open connection exists
			"open": bson.M{"$gt": []interface{}{bson.M{"$ifNull": []interface{}{"$open_duration", 0}}, 0}},
		}},
		bson.M{"$unwind": "$maxdur"},
		bson.M{"$unwind": "$tuples"},
		bson.M{"$unwind": "$tuples"}, // not an error, must be done twice
		bson.M{"$group": bson.M{
			"_id":              "$_id",
			"maxdur":           bson.M{"$max": "$maxdur"},
			"src":              bson.M{"$first": "$src"},
			"src_network_uuid": bson.M{"$first": "$src_network_uuid"},
			"src_network_name": bson.M{"$first": "$src_network_name"},
			"dst":              bson.M{"$first": "$dst"},
			"dst_network_uuid": bson.M{"$first": "$dst_network_uuid"},
			"dst_network_name": bson.M{"$first": "$dst_network_name"},
			"tuples":           bson.M{"$addToSet": "$tuples"},
			"open":             bson.M{"$first": "$open"},
		}},
		bson.M{"$project": bson.M{
			"maxdur":           1,
			"src":              1,
			"src_network_uuid": 1,
			"src_network_name": 1,
			"dst":              1,
			"dst_network_uuid": 1,
			"dst_network_name": 1,
			"tuples":           bson.M{"$slice": []interface{}{"$tuples", 5}},
			"open":             1,
		}},
		bson.M{"$sort": bson.M{"maxdur": -1}},
	}

	if !noLimit {
		longConnQuery = append(longConnQuery, bson.M{"$limit": limit})
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).Pipe(longConnQuery).AllowDiskUse().All(&longConnResults)

	return longConnResults, err

}
