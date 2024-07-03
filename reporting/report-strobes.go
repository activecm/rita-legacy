package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/activecm/rita-legacy/pkg/beacon"
	"github.com/activecm/rita-legacy/reporting/templates"
	"github.com/activecm/rita-legacy/resources"
)

func printStrobes(db string, showNetNames bool, res *resources.Resources, logsGeneratedAt string) error {
	f, err := os.Create("strobes.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var strobesTempl string
	if showNetNames {
		strobesTempl = templates.StrobesNetNamesTempl
	} else {
		strobesTempl = templates.StrobesTempl
	}

	out, err := template.New("strobes.html").Parse(strobesTempl)
	if err != nil {
		return err
	}

	data, err := beacon.StrobeResults(res, -1, 1000, false)
	if err != nil {
		return err
	}

	w, err := getStrobesWriter(data, showNetNames)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w), LogsGeneratedAt: logsGeneratedAt})
}

func getStrobesWriter(strobes []beacon.StrobeResult, showNetNames bool) (string, error) {
	var tmpl string
	if showNetNames {
		tmpl = "<tr><td>{{.SrcNetworkName}}</td><td>{{.DstNetworkName}}</td><td>{{.SrcIP}}</td><td>{{.DstIP}}</td><td>{{.ConnectionCount}}</td></tr>\n"
	} else {
		tmpl = "<tr><td>{{.SrcIP}}</td><td>{{.DstIP}}</td><td>{{.ConnectionCount}}</td></tr>\n"
	}

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
