package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/activecm/rita/datatypes/strobe"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
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

	var strobes []strobe.Strobe
	coll := res.DB.Session.DB(db).C(res.Config.T.Strobe.StrobeTable)
	coll.Find(nil).Sort("-connection_count").Limit(1000).All(&strobes)

	w, err := getStrobesWriter(strobes)
	if err != nil {
		return err
	}
	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getStrobesWriter(strobes []strobe.Strobe) (string, error) {
	tmpl := "<tr><td>{{.Source}}</td><td>{{.Destination}}</td><td>{{.ConnectionCount}}</td></tr>\n"
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
