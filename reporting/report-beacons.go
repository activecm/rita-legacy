package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/activecm/rita-legacy/pkg/beacon"
	"github.com/activecm/rita-legacy/reporting/templates"
	"github.com/activecm/rita-legacy/resources"
)

func printBeacons(db string, showNetNames bool, res *resources.Resources, logsGeneratedAt string) error {
	var w string
	f, err := os.Create("beacons.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var beaconsTempl string
	if showNetNames {
		beaconsTempl = templates.BeaconsNetNamesTempl
	} else {
		beaconsTempl = templates.BeaconsTempl
	}

	out, err := template.New("beacon.html").Parse(beaconsTempl)
	if err != nil {
		return err
	}

	data, err := beacon.Results(res, 0)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		w = ""
	} else {
		w, err = getBeaconWriter(data, showNetNames)
		if err != nil {
			return err
		}
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w), LogsGeneratedAt: logsGeneratedAt})
}

func getBeaconWriter(beacons []beacon.Result, showNetNames bool) (string, error) {
	tmpl := "<tr>"

	tmpl += "<td>{{printf \"%.3f\" .Score}}</td>"

	if showNetNames {
		tmpl += "<td>{{.SrcNetworkName}}</td><td>{{.DstNetworkName}}</td><td>{{.SrcIP}}</td><td>{{.DstIP}}</td>"
	} else {
		tmpl += "<td>{{.SrcIP}}</td><td>{{.DstIP}}</td>"
	}
	tmpl += "<td>{{.Connections}}</td><td>{{printf \"%.3f\" .AvgBytes}}</td><td>{{.TotalBytes}}</td><td>{{printf \"%.3f\" .Ts.Score}}</td>"
	tmpl += "<td>{{printf \"%.3f\" .Ds.Score}}</td><td>{{printf \"%.3f\" .DurScore}}</td><td>{{printf \"%.3f\" .HistScore}}</td><td>{{.Ts.Mode}}</td>"
	tmpl += "</tr>\n"

	out, err := template.New("beacon").Parse(tmpl)
	if err != nil {
		return "", err
	}

	w := new(bytes.Buffer)

	for _, result := range beacons {
		err = out.Execute(w, result)
		if err != nil {
			return "", err
		}
	}

	return w.String(), nil
}
