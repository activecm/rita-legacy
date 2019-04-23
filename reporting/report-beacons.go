package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/activecm/rita/pkg/beacon"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/urfave/cli"
)

func printBeacons(db string, res *resources.Resources) error {
	f, err := os.Create("beacons.html")
	if err != nil {
		return err
	}
	defer f.Close()

	out, err := template.New("beacon.html").Parse(templates.BeaconsTempl)
	if err != nil {
		return err
	}

	data := getBeaconResultsView(res, 0)

	if !(len(data) > 0) {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	w, err := getBeaconWriter(data)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getBeaconWriter(beacons []beacon.AnalysisView) (string, error) {
	tmpl := "<tr><td>{{printf \"%.3f\" .Score}}</td><td>{{.Src}}</td><td>{{.Dst}}</td><td>{{.Connections}}</td><td>{{printf \"%.3f\" .AvgBytes}}</td><td>"
	tmpl += "{{.Ts.Range}}</td><td>{{.Ds.Range}}</td><td>{{.Ts.Mode}}</td><td>{{.Ds.Mode}}</td><td>{{.Ts.ModeCount}}</td><td>{{.Ds.ModeCount}}<td>"
	tmpl += "{{printf \"%.3f\" .Ts.Skew}}</td><td>{{printf \"%.3f\" .Ds.Skew}}</td><td>{{.Ts.Dispersion}}</td><td>{{.Ds.Dispersion}}</td>"
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

//getBeaconResultsView finds beacons greater than a given cutoffScore
//and links the data from the unique connections table back in to the results
func getBeaconResultsView(res *resources.Resources, cutoffScore float64) []beacon.AnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var beacons []beacon.AnalysisView

	beaconQuery := bson.M{"score": bson.M{"$gt": cutoffScore}}

	//TODO: Don't swallow this error
	_ = ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Beacon.BeaconTable).Find(beaconQuery).Sort("-score").All(&beacons)

	return beacons

}
