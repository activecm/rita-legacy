package printing

import (
	"bytes"
	"html/template"
	"os"
	"strconv"

	"github.com/bglebrun/rita/database"
	"github.com/bglebrun/rita/datatypes/scanning"
	"github.com/bglebrun/rita/printing/templates"
	"github.com/olekukonko/tablewriter"
)

type scan struct {
	Dbs    string
	Writer string
}

func printScans(res *database.Resources, db string, dir string) error {
	var scans []scanning.Scan
	coll := res.DB.Session.DB(db).C(res.System.ScanningConfig.ScanTable)
	coll.Find(nil).All(&scans)

	return printScansHTML(scans, db, dir)

}

// printScansHTML prints all scans for a given database
func printScansHTML(scans []scanning.Scan, db string, dir string) error {
	f, err := os.Create(dir + "scans.html")
	if err != nil {
		return err
	}
	defer f.Close()

	w := new(bytes.Buffer)

	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Source", "Destination", "Ports Scanned"})
	for _, scan := range scans {
		table.Append([]string{scan.Src, scan.Dst, strconv.Itoa(scan.PortCount)})
	}
	table.Render()

	out, err := template.New("scans.html").Parse(templates.ScansTempl)
	if err != nil {
		return err
	}

	return out.Execute(f, &scan{Dbs: db, Writer: w.String()})
}
