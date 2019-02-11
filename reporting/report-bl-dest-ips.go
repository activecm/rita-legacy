package reporting

import (
	"html/template"
	"os"

	"github.com/globalsign/mgo/bson"

	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
)

func printBLDestIPs(db string, res *resources.Resources) error {
	f, err := os.Create("bl-dest-ips.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var blIPs []host.AnalysisView

	blacklistFindQuery := bson.M{
		"$and": []bson.M{
			bson.M{"blacklisted": true},
			bson.M{"count_dst": bson.M{"$gt": 0}},
		}}

	res.DB.Session.DB(db).
		C(res.Config.T.Structure.HostTable).
		Find(blacklistFindQuery).Sort("-conn").All(&blIPs)

	for i, entry := range blIPs {
		var connected []uconn.AnalysisView
		res.DB.Session.DB(db).
			C(res.Config.T.Structure.UniqueConnTable).Find(
			bson.M{"dst": entry.Host},
		).All(&connected)
		for _, uconn := range connected {
			blIPs[i].ConnectedHosts = append(blIPs[i].ConnectedHosts, uconn.Src)
		}
	}

	out, err := template.New("bl-dest-ips.html").Parse(templates.BLDestIPTempl)
	if err != nil {
		return err
	}

	w, err := getBLIPWriter(blIPs)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}
