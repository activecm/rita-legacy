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
)

func printBLHostnames(db string, res *resources.Resources) error {
	f, err := os.Create("bl-hostnames.html")
	if err != nil {
		return err
	}
	defer f.Close()

	data := getBlacklistedHostnameResultsView(res, 0, "conn_count")

	out, err := template.New("bl-hostnames.html").Parse(templates.BLHostnameTempl)
	if err != nil {
		return err
	}

	w, err := getBLHostnameWriter(data)
	if err != nil {
		return err
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

//getBeaconResultsView finds beacons greater than a given cutoffScore
//and links the data from the unique connections table back in to the results
func getBlacklistedHostnameResultsView(res *resources.Resources, cutoffScore float64, sort string) []hostname.AnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	blHostsQuery := []bson.M{
		bson.M{"$match": bson.M{"blacklisted": true}},
		bson.M{"$unwind": "$dat"},
		bson.M{"$project": bson.M{"host": 1, "ip": "$dat.ips"}},
		bson.M{"$unwind": "$ip"},
		bson.M{"$lookup": bson.M{
			"from":         "uconn",
			"localField":   "ip",
			"foreignField": "dst",
			"as":           "uconn",
		}},
		bson.M{"$unwind": "$uconn"},
		bson.M{"$group": bson.M{
			"_id":         "$host",
			"host":        bson.M{"$first": "$host"},
			"total_bytes": bson.M{"$sum": "$uconn.total_bytes"},
			"conn_count":  bson.M{"$sum": "$uconn.connection_count"},
			"uconn_count": bson.M{"$sum": 1},
			"srcs":        bson.M{"$push": "$uconn.src"},
		}},
		bson.M{"$sort": bson.M{sort: -1}},
	}

	var blHosts []hostname.AnalysisView

	_ = ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.HostnamesTable).Pipe(blHostsQuery).All(&blHosts)

	return blHosts

}
