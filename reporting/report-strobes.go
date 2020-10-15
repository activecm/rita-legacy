package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/activecm/rita/pkg/beacon"
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

	data, err := beacon.StrobeResults(res, -1, 1000, false)
	if err != nil {
		return err
	}

	w, err := getStrobesWriter(data)
	if err != nil {
		return err
	}
	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getStrobesWriter(strobes []beacon.StrobeResult) (string, error) {
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
