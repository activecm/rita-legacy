package reporting

import (
	"bytes"
	"html/template"
	"os"
	"sort"

	"gopkg.in/mgo.v2/bson"

	"github.com/activecm/rita/datatypes/blacklist"
	"github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
)

func printBLSourceIPs(db string, res *resources.Resources) error {
	f, err := os.Create("bl-source-ips.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var blIPs []blacklist.BlacklistedIP
	res.DB.Session.DB(db).
		C(res.Config.T.Blacklisted.SourceIPsTable).
		Find(nil).Sort("-conn").All(&blIPs)

	for i, ip := range blIPs {
		var connected []structure.UniqueConnection
		res.DB.Session.DB(db).
			C(res.Config.T.Structure.UniqueConnTable).Find(
			bson.M{"src": ip.IP},
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

func getBLIPWriter(results []blacklist.BlacklistedIP) (string, error) {
	tmpl := "<tr><td>{{.IP}}</td><td>{{.Connections}}</td><td>{{.UniqueConnections}}</td>" +
		"<td>{{.TotalBytes}}</td>" +
		"<td>{{range $idx, $list := .Lists}}{{if $idx}}, {{end}}{{ $list }}{{end}}</td>" +
		"<td>{{range $idx, $host := .ConnectedHosts}}{{if $idx}}, {{end}}{{ $host }}{{end}}</td>" +
		"</tr>\n"

	out, err := template.New("blip").Parse(tmpl)
	if err != nil {
		return "", err
	}

	w := new(bytes.Buffer)

	for _, result := range results {
		sort.Strings(result.Lists)
		sort.Strings(result.ConnectedHosts)
		err := out.Execute(w, result)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}
