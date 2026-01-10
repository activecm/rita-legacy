package uconn

import (
	"github.com/activecm/rita-legacy/resources"
	"github.com/globalsign/mgo/bson"
)

// LongConnResults returns long connections longer than the given thresh in
// seconds. The results will be sorted, descending by duration.
// limit and noLimit control how many results are returned.
func LongConnResults(res *resources.Resources, thresh int, limit int, noLimit bool) ([]LongConnResult, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var longConnResults []LongConnResult

	longConnQuery := []bson.M{
		{"$match": bson.M{"dat.maxdur": bson.M{"$gt": thresh}}},
		{"$project": bson.M{
			"src":              1,
			"src_network_uuid": 1,
			"src_network_name": 1,
			"dst":              1,
			"dst_network_uuid": 1,
			"dst_network_name": 1,
			"count":            1,
			"tbytes":           1,
			"tdur":             1,
			"maxdur":           "$dat.maxdur",
			"tuples":           bson.M{"$ifNull": []interface{}{"$dat.tuples", []interface{}{}}},
			"open":             1,
		}},
		{"$unwind": "$maxdur"},
		{"$unwind": "$tuples"},
		{"$unwind": "$tuples"}, // not an error, must be done twice
		{"$group": bson.M{
			"_id":              "$_id",
			"maxdur":           bson.M{"$max": "$maxdur"},
			"src":              bson.M{"$first": "$src"},
			"src_network_uuid": bson.M{"$first": "$src_network_uuid"},
			"src_network_name": bson.M{"$first": "$src_network_name"},
			"dst":              bson.M{"$first": "$dst"},
			"dst_network_uuid": bson.M{"$first": "$dst_network_uuid"},
			"dst_network_name": bson.M{"$first": "$dst_network_name"},
			"count":            bson.M{"$first": "$count"},
			"tbytes":           bson.M{"$first": "$tbytes"},
			"tdur":             bson.M{"$first": "$tdur"},
			"tuples":           bson.M{"$addToSet": "$tuples"},
			"open":             bson.M{"$first": "$open"},
		}},
		{"$project": bson.M{
			"maxdur":           1,
			"src":              1,
			"src_network_uuid": 1,
			"src_network_name": 1,
			"dst":              1,
			"dst_network_uuid": 1,
			"dst_network_name": 1,
			"count":            1,
			"tbytes":           1,
			"tdur":             1,
			"tuples":           bson.M{"$slice": []interface{}{"$tuples", 5}},
			"open":             1,
		}},
		{"$sort": bson.M{"tdur": -1, "maxdur": -1}},
	}

	if !noLimit {
		longConnQuery = append(longConnQuery, bson.M{"$limit": limit})
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).Pipe(longConnQuery).AllowDiskUse().All(&longConnResults)

	return longConnResults, err

}

// OpenConnResults returns open connections. The results will be sorted, descending by duration.
// limit and noLimit control how many results are returned.
func OpenConnResults(res *resources.Resources, thresh int, limit int, noLimit bool) ([]OpenConnResult, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var openConnResults []OpenConnResult

	openConnQuery := []bson.M{
		{"$match": bson.M{"open": true}},
		{"$project": bson.M{
			"dst":              1,
			"dst_network_name": 1,
			"dst_network_uuid": 1,
			"src":              1,
			"src_network_name": 1,
			"src_network_uuid": 1,
			"open_conns":       bson.M{"$objectToArray": "$open_conns"},
		}},
		{"$unwind": "$open_conns"},
		{"$project": bson.M{
			"_id":              0,
			"dst":              1,
			"dst_network_name": 1,
			"dst_network_uuid": 1,
			"src":              1,
			"src_network_name": 1,
			"src_network_uuid": 1,
			// we have to use the .v. here because the objectToArray operator creates a k and v fields
			// that hold the object's key (which was the UID in this case)
			// and the object's values, respectively
			"bytes":    "$open_conns.v.bytes",
			"duration": "$open_conns.v.duration",
			"tuple":    "$open_conns.v.tuple",
			"uid":      "$open_conns.k",
		}},
		{"$sort": bson.M{"duration": -1}},
	}

	if !noLimit {
		openConnQuery = append(openConnQuery, bson.M{"$limit": limit})
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).Pipe(openConnQuery).AllowDiskUse().All(&openConnResults)

	return openConnResults, err

}
