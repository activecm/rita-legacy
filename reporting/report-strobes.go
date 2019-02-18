package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/activecm/rita/pkg/beacon"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
)

func printStrobes(db string, res *resources.Resources) error {
	f, err := os.Create("strobes.html")
	if err != nil {
		return err
	}
	defer f.Close()
	out, err := template.New("strobes.html").Parse(templates.StrobesTempl)
	if err != nil {
		return err
	}

	data := getStrobeResultsView(res)

	w, err := getStrobesWriter(data)
	if err != nil {
		return err
	}
	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getStrobesWriter(strobes []beacon.StrobeAnalysisView) (string, error) {
	tmpl := "<tr><td>{{.Src}}</td><td>{{.Dst}}</td><td>{{.ConnectionCount}}</td></tr>\n"
	out, err := template.New("Strobes").Parse(tmpl)
	if err != nil {
		return "", err
	}
	w := new(bytes.Buffer)
	for _, strobe := range strobes {
		err := out.Execute(w, strobe)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}

//getStrobeResultsView ...
func getStrobeResultsView(res *resources.Resources) []beacon.StrobeAnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var strobes []beacon.StrobeAnalysisView

	strobeQuery := bson.M{"strobe": true}

	_ = ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).Find(strobeQuery).Sort("-connection_count").Limit(1000).All(&strobes)

	return strobes

}
