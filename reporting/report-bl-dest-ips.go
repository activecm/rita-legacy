package reporting

import (
	"html/template"
	"os"

	"github.com/activecm/rita-legacy/pkg/blacklist"
	"github.com/activecm/rita-legacy/reporting/templates"
	"github.com/activecm/rita-legacy/resources"
)

func printBLDestIPs(db string, showNetNames bool, res *resources.Resources, logsGeneratedAt string) error {
	f, err := os.Create("bl-dest-ips.html")
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := blacklist.DstIPResults(res, "conn_count", 1000, false)
	if err != nil {
		return err
	}

	var blDestIPTempl string
	if showNetNames {
		blDestIPTempl = templates.BLDestIPNetNamesTempl
	} else {
		blDestIPTempl = templates.BLDestIPTempl
	}

	out, err := template.New("bl-dest-ips.html").Parse(blDestIPTempl)
	if err != nil {
		return err
	}

	w, err := getBLIPWriter(data, showNetNames)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w), LogsGeneratedAt: logsGeneratedAt})
}
