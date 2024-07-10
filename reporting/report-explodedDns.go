package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/activecm/rita-legacy/pkg/explodeddns"
	"github.com/activecm/rita-legacy/reporting/templates"
	"github.com/activecm/rita-legacy/resources"
)

func printDNS(db string, showNetNames bool, res *resources.Resources, logsGeneratedAt string) error {
	f, err := os.Create("dns.html")
	if err != nil {
		return err
	}
	defer f.Close()

	res.DB.SelectDB(db)

	limit := 1000

	data, err := explodeddns.Results(res, limit, false)
	if err != nil {
		return err
	}

	out, err := template.New("dns.html").Parse(templates.DNStempl)
	if err != nil {
		return err
	}

	w, err := getDNSWriter(data)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w), LogsGeneratedAt: logsGeneratedAt})
}

func getDNSWriter(results []explodeddns.Result) (string, error) {
	tmpl := "<tr><td>{{.SubdomainCount}}</td><td>{{.Visited}}</td><td>{{.Domain}}</td></tr>\n"

	out, err := template.New("dns").Parse(tmpl)
	if err != nil {
		return "", err
	}

	w := new(bytes.Buffer)

	for _, result := range results {
		err := out.Execute(w, result)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}
