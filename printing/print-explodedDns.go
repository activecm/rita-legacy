package printing

import (
	"bytes"
	"html/template"
	"os"

	"github.com/bglebrun/rita/database"
	"github.com/bglebrun/rita/datatypes/dns"
	"github.com/bglebrun/rita/printing/templates"
)

func printDNSHtml(db string, res *database.Resources) error {
	f, err := os.Create("dns.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var results []dns.ExplodedDNS
	iter := res.DB.Session.DB(db).C(res.System.DnsConfig.ExplodedDnsTable).Find(nil)
	iter.Sort("-subdomains").All(&results)
	out, err := template.New("dns.html").Parse(templates.DNStempl)
	if err != nil {
		return err
	}

	w, err := getDNSWriter(results)
	if err != nil {
		return err
	}

	return out.Execute(f, &scan{Dbs: db, Writer: template.HTML(w)})
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
