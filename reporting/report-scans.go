package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/scanning"
	"github.com/ocmdev/rita/reporting/templates"
)

type scan struct {
	Dbs    string
	Writer template.HTML
}

func printScans(db string, res *database.Resources) error {
	f, err := os.Create("scans.html")
	if err != nil {
		return err
	}
	defer f.Close()

	out, err := template.New("scans.html").Parse(templates.ScansTempl)
	if err != nil {
		return err
	}

	var scans []scanning.Scan
	coll := res.DB.Session.DB(db).C(res.System.ScanningConfig.ScanTable)
	coll.Find(nil).All(&scans)

	w, err := getScanWriter(scans)
	if err != nil {
		return err
	}

	return out.Execute(f, &scan{Dbs: db, Writer: template.HTML(w)})
}

func getScanWriter(scans []scanning.Scan) (string, error) {

	tmpl := "<tr><td>{{.Src}}</td><td>{{.Dst}}</td><td>{{.PortCount}}</td><td>{{range $idx, $port := .PortSet}}{{if $idx}}{{end}} -- {{ $port }}{{end}} -- </td></tr>\n"

	out, err := template.New("scn").Parse(tmpl)
	if err != nil {
		return "", err
	}

	w := new(bytes.Buffer)

	for _, scan := range scans {
		err := out.Execute(w, scan)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}
