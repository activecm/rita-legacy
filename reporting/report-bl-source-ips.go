package reporting

import (
	"bytes"
	"html/template"
	"os"
	"sort"

	"github.com/globalsign/mgo/bson"

	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
	"github.com/urfave/cli"
)

func printBLSourceIPs(db string, res *resources.Resources) error {
	f, err := os.Create("bl-source-ips.html")
	if err != nil {
		return err
	}
	defer f.Close()

	match := bson.M{
		"$and": []bson.M{
			bson.M{"blacklisted": true},
			bson.M{"dat.count_src": bson.M{"$gt": 0}},
		}}

	data := getBlacklistedIPsResultsView(res, "conn_count", 1000, match, "src", "dst")

	out, err := template.New("bl-source-ips.html").Parse(templates.BLSourceIPTempl)
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
