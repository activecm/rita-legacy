package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/activecm/rita/pkg/explodeddns"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
	"github.com/urfave/cli"
)

func printDNS(db string, res *resources.Resources) error {
	f, err := os.Create("dns.html")
	if err != nil {
		return err
	}
	defer f.Close()

	res.DB.SelectDB(db)

	limit := 1000

	data := getExplodedDNSResultsView(res, limit)

	out, err := template.New("dns.html").Parse(templates.DNStempl)
	if err != nil {
		return err
	}

	w, err := getDNSWriter(data)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getDNSWriter(results []explodeddns.AnalysisView) (string, error) {
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

//getExplodedDNSResultsView gets the exploded dns results
func getExplodedDNSResultsView(res *resources.Resources, limit int) []explodeddns.AnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var explodedDNSResults []explodeddns.AnalysisView

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.ExplodedDNSTable).Find(nil).Sort("-subdomain_count").Limit(limit).All(&explodedDNSResults)

	if err != nil {
		cli.NewExitError(err.Error(), -1)
	}

	return explodedDNSResults

}
