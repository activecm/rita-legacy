package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/ocmdev/rita/analysis/beacon"
	"github.com/ocmdev/rita/database"
	beaconData "github.com/ocmdev/rita/datatypes/beacon"
	"github.com/ocmdev/rita/reporting/templates"
)

func printBeacons(db string, res *database.Resources) error {
	f, err := os.Create("beacons.html")
	if err != nil {
		return err
	}
	defer f.Close()

	out, err := template.New("beacon.html").Parse(templates.BeaconsTempl)
	if err != nil {
		return err
	}

	res.DB.SelectDB(db)
	var data []beaconData.BeaconAnalysisView
	ssn := res.DB.Session.Copy()
	beacon.GetBeaconResultsView(res, ssn, 0).All(&data)
	ssn.Close()

	w, err := getBeaconWriter(data)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getBeaconWriter(beacons []beaconData.BeaconAnalysisView) (string, error) {
	tmpl := "<tr><td>{{printf \"%.3f\" .TS_score}}</td><td>{{.Src}}</td><td>{{.Dst}}</td><td>{{.Connections}}</td><td>{{printf \"%.3f\" .AvgBytes}}</td><td>"
	tmpl += "{{.TS_iRange}}</td><td>{{.TS_iMode}}</td><td>{{.TS_iModeCount}}</td><td>"
	tmpl += "{{printf \"%.3f\" .TS_iSkew}}</td><td>{{.TS_iDispersion}}</td><td>{{printf \"%.3f\" .TS_duration}}</tr>\n"

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
