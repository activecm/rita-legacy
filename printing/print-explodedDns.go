package printing

import (
	"bytes"
	"html/template"
	"os"
	"strconv"

	"github.com/bglebrun/rita/printing/templates"
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/dns"
	"github.com/olekukonko/tablewriter"
)

func printDNSHtml(db string, res *database.Resources, dir string) error {
	f, err := os.Create(dir + "/dns.html")
	if err != nil {
		return err
	}
	defer f.Close()

	w := new(bytes.Buffer)

	var explodedResults []dns.ExplodedDNS
	iter := res.DB.Session.DB(db).C(res.System.DnsConfig.ExplodedDnsTable).Find(nil)
	count, _ := iter.Count()

	iter.Sort("-subdomains").Limit(count).All(&explodedResults)

	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Domain", "Unique Subdomains", "Times Looked Up"})
	for _, result := range explodedResults {
		table.Append([]string{
			result.Domain,
			strconv.FormatInt(result.Subdomains, 10),
			strconv.FormatInt(result.Visited, 10),
		})
	}
	table.Render()
	out, err := template.New("dns.html").Parse(templates.DNStempl)
	if err != nil {
		return err
	}

	return out.Execute(f, &scan{Dbs: db, Writer: w.String()})
}
