package reporting

import (
	"html/template"
	"os"

	"gopkg.in/mgo.v2/bson"

	"github.com/activecm/rita/datatypes/blacklist"
	"github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
)

func printBLDestIPs(db string, res *resources.Resources) error {
	f, err := os.Create("bl-dest-ips.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var blIPs []blacklist.BlacklistedIP
	res.DB.Session.DB(db).
		C(res.Config.T.Blacklisted.DestIPsTable).
		Find(nil).Sort("-conn").All(&blIPs)

	for i, ip := range blIPs {
		var connected []structure.UniqueConnection
		res.DB.Session.DB(db).
			C(res.Config.T.Structure.UniqueConnTable).Find(
			bson.M{"dst": ip.IP},
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
