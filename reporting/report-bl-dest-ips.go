package reporting

import (
	"html/template"
	"os"

	"github.com/globalsign/mgo/bson"

	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
	"github.com/urfave/cli"
)

func printBLDestIPs(db string, res *resources.Resources) error {
	f, err := os.Create("bl-dest-ips.html")
	if err != nil {
		return err
	}
	defer f.Close()

	match := bson.M{
		"$and": []bson.M{
			bson.M{"blacklisted": true},
			bson.M{"dat.count_dst": bson.M{"$gt": 0}},
		}}

	data := getBlacklistedIPsResultsView(res, "conn_count", 1000, match, "dst", "src")

	out, err := template.New("bl-dest-ips.html").Parse(templates.BLDestIPTempl)
	if err != nil {
		return err
	}

	w, err := getBLIPWriter(data)
	if err != nil {
		return err
	}
	if len(w) == 0 {
		return cli.NewExitError("No results were found for " + db, -1)
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

//getBlaclistedIPsResultsView
func getBlacklistedIPsResultsView(res *resources.Resources, sort string, limit int, match bson.M, field1 string, field2 string) []host.AnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var blIPs []host.AnalysisView

	blIPQuery := []bson.M{
		bson.M{"$match": match},
		bson.M{"$project": bson.M{"host": "$ip"}},
		bson.M{"$lookup": bson.M{
			"from":         "uconn",
			"localField":   "host",
			"foreignField": field1,
			"as":           "u",
		}},
		bson.M{"$unwind": "$u"},
		bson.M{"$unwind": "$u.dat"},
		bson.M{"$project": bson.M{"host": 1, "conns": "$u.dat.count", "bytes": "$u.dat.tbytes", "ip": ("$u." + field2)}},
		bson.M{"$group": bson.M{
			"_id":         "$host",
			"host":        bson.M{"$first": "$host"},
			"ips":         bson.M{"$addToSet": "$ip"},
			"conn_count":  bson.M{"$sum": "$conns"},
			"total_bytes": bson.M{"$sum": "$bytes"},
		}},
		bson.M{"$sort": bson.M{sort: -1}},
		bson.M{"$limit": limit},
		bson.M{"$project": bson.M{
			"_id":         0,
			"uconn_count": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$ips", []interface{}{}}}},
			"ips":         1,
			"conn_count":  1,
			"host":        1,
			"total_bytes": 1,
		}},
	}

	//TODO: Don't swallow this error
	_ = ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.HostTable).Pipe(blIPQuery).AllowDiskUse().All(&blIPs)

	return blIPs

}
