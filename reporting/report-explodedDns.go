package reporting

import (
	"bytes"
	"html/template"
	"os"
	"strconv"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/dns"
	"github.com/ocmdev/rita/reporting/templates"
	"github.com/fatih/color"
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

	if len(results) > 100000 {
		color.Red("[!!] WARNING: Database " + db + " has a VERY large DNS page (" + strconv.Itoa(len(results)) + " results written)")
		color.Red("[!!] May crash your browser, consider using something like \"grep -v\" to filter or plaintext to view these results")
	} else if len(results) > 9000 {
		color.Yellow("[-] WARNING: Database " + db + " has a large DNS page (" + strconv.Itoa(len(results)) + " results written)")
		color.Yellow("[-] Page may be slow to load in some browsers")
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
