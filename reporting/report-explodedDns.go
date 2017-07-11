package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/dns"
	"github.com/ocmdev/rita/reporting/templates"
)

func printDNS(db string, res *database.Resources) error {
	f, err := os.Create("dns.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var results []dns.ExplodedDNS
	iter := res.DB.Session.DB(db).C(res.Config.T.DNS.ExplodedDNSTable).Find(nil)
	iter.Sort("-subdomains").Limit(1000).All(&results)

	out, err := template.New("dns.html").Parse(templates.DNStempl)
	if err != nil {
		return err
	}

	w, err := getDNSWriter(results)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getDNSWriter(results []dns.ExplodedDNS) (string, error) {
	tmpl := "<tr><td>{{.Subdomains}}</td><td>{{.Visited}}</td><td>{{.Domain}}</td></tr>\n"

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
