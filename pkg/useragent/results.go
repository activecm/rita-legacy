package useragent

import (
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
)

//Results returns useragents sorted by how many times each useragent was
//seen in the dataset. sortDirection controls where the useragents are
//sorted in descending (sortDirection=-1) or ascending order (sortDirection=1).
//limit and noLimit control how many results are returned.
func Results(res *resources.Resources, sortDirection, limit int, noLimit bool) ([]Result, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var useragentResults []Result

	useragentQuery := []bson.M{
		{"$project": bson.M{"user_agent": 1, "seen": "$dat.seen"}},
		{"$unwind": "$seen"},
		{"$group": bson.M{
			"_id":  "$user_agent",
			"seen": bson.M{"$sum": "$seen"},
		}},
		{"$project": bson.M{
			"_id":        0,
			"user_agent": "$_id",
			"seen":       1,
		}},
		{"$sort": bson.M{"seen": sortDirection}},
	}

	if !noLimit {
		useragentQuery = append(useragentQuery, bson.M{"$limit": limit})
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.UserAgent.UserAgentTable).Pipe(useragentQuery).AllowDiskUse().All(&useragentResults)

	return useragentResults, err

}
