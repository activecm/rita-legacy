package reporting

import (
	"bytes"
	"html/template"
	"os"
	"sort"

	"github.com/globalsign/mgo/bson"

	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
	"github.com/urfave/cli"
)

func printBLHostnames(db string, res *resources.Resources) error {
	f, err := os.Create("bl-hostnames.html")
	if err != nil {
		return err
	}
	defer f.Close()

	data := getBlacklistedHostnameResultsView(res, "conn_count", 1000)

	out, err := template.New("bl-hostnames.html").Parse(templates.BLHostnameTempl)
	if err != nil {
		return err
	}

	w, err := getBLHostnameWriter(data)
	if err != nil {
		return err
	}
	if len(w) == 0 {
		return cli.NewExitError("No results were found for " + db, -1)
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getBLHostnameWriter(results []hostname.AnalysisView) (string, error) {
	tmpl := "<tr><td>{{.Host}}</td><td>{{.Connections}}</td><td>{{.UniqueConnections}}</td>" +
		"<td>{{.TotalBytes}}</td>" +
		"<td>{{range $idx, $host := .ConnectedHosts}}{{if $idx}}, {{end}}{{ $host }}{{end}}</td>" +
		"</tr>\n"

	out, err := template.New("blhostname").Parse(tmpl)
	if err != nil {
		return "", err
	}

	w := new(bytes.Buffer)

	for _, result := range results {
		sort.Strings(result.ConnectedHosts)
		err := out.Execute(w, result)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}

//getBlacklistedHostnameResultsView ....
func getBlacklistedHostnameResultsView(res *resources.Resources, sort string, limit int) []hostname.AnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	blHostsQuery := []bson.M{
		bson.M{"$match": bson.M{"blacklisted": true}},
		bson.M{"$unwind": "$dat"},
		bson.M{"$project": bson.M{"host": 1, "ips": "$dat.ips"}},
		bson.M{"$unwind": "$ips"},
		bson.M{"$group": bson.M{
			"_id": "$host",
			"ips": bson.M{"$addToSet": "$ips"},
		}},
		bson.M{"$unwind": "$ips"},
		bson.M{"$lookup": bson.M{
			"from":         "uconn",
			"localField":   "ips",
			"foreignField": "dst",
			"as":           "uconn",
		}},
		bson.M{"$unwind": "$uconn"},
		bson.M{"$unwind": "$uconn.dat"},
		bson.M{"$project": bson.M{"host": 1, "conns": "$uconn.dat.count", "bytes": "$uconn.dat.tbytes", "ip": "$uconn.src"}},
		bson.M{"$group": bson.M{
			"_id":         "$_id",
			"ips":         bson.M{"$addToSet": "$ip"},
			"conn_count":  bson.M{"$sum": "$conns"},
			"total_bytes": bson.M{"$sum": "$bytes"},
		}},
		bson.M{"$sort": bson.M{sort: -1}},
		bson.M{"$limit": limit},
		bson.M{"$project": bson.M{
			"_id":         0,
			"host":        "$_id",
			"uconn_count": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$ips", []interface{}{}}}},
			"ips":         1,
			"conn_count":  1,
			"total_bytes": 1,
		}},
	}

	var blHosts []hostname.AnalysisView

	//TODO: Don't swallow this error
	_ = ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.HostnamesTable).Pipe(blHostsQuery).AllowDiskUse().All(&blHosts)

	return blHosts

}
