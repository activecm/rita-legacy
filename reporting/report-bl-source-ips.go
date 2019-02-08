package reporting

import (
	"bytes"
	"html/template"
	"os"
	"sort"

	"github.com/globalsign/mgo/bson"

	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
)

func printBLSourceIPs(db string, res *resources.Resources) error {
	f, err := os.Create("bl-source-ips.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var blIPs []host.AnalysisView

	blacklistFindQuery := bson.M{
		"$and": []bson.M{
			bson.M{"blacklisted": true},
			bson.M{"count_src": bson.M{"$gt": 0}},
		}}

	res.DB.Session.DB(db).
		C(res.Config.T.Structure.HostTable).
		Find(blacklistFindQuery).Sort("-conn").All(&blIPs)

	for i, entry := range blIPs {
		var connected []uconn.AnalysisView
		res.DB.Session.DB(db).
			C(res.Config.T.Structure.UniqueConnTable).Find(
			bson.M{"src": entry.Host},
		).All(&connected)
		for _, uconn := range connected {
			blIPs[i].ConnectedHosts = append(blIPs[i].ConnectedHosts, uconn.Dst)
		}
	}

	out, err := template.New("bl-source-ips.html").Parse(templates.BLSourceIPTempl)
	if err != nil {
		return err
	}

	w, err := getBLIPWriter(blIPs)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getBLIPWriter(results []host.AnalysisView) (string, error) {
	tmpl := "<tr><td>{{.Host}}</td><td>{{.Connections}}</td><td>{{.UniqueConnections}}</td>" +
		"<td>{{.TotalBytes}}</td>" +
		"<td>{{range $idx, $host := .ConnectedHosts}}{{if $idx}}, {{end}}{{ $host }}{{end}}</td>" +
		"</tr>\n"

	out, err := template.New("blip").Parse(tmpl)
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
