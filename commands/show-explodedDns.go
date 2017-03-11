package commands

import (
	"fmt"
	"os"
	"strconv"
	"text/template"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/dns"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:  "show-exploded-dns",
		Usage: "Print dns analysis. Exposes covert dns channels.",
		Flags: []cli.Flag{
			humanFlag,
			databaseFlag,
			allFlag,
		},
		Action: func(c *cli.Context) error {
			if c.String("database") == "" {
				return cli.NewExitError("Specify a database with -d", -1)
			}

			res := database.InitResources("")

			var explodedResults []dns.ExplodedDNS
			iter := res.DB.Session.DB(c.String("database")).C(res.System.DnsConfig.ExplodedDnsTable).Find(nil)
			count, _ := iter.Count()

			if !c.Bool("all") {
				count = 15
			}

			iter.Sort("-subdomains").Limit(count).All(&explodedResults)

			if c.Bool("human-readable") {
				return showResultsHuman(explodedResults)
			}
			return showResults(explodedResults)
		},
	}
	bootstrapCommands(command)
}

func showResults(dnsResults []dns.ExplodedDNS) error {
	tmpl := "{{.Domain}},{{.Subdomains}},{{.Visited}}\n"

	out, err := template.New("exploded-dns").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	var error error = nil
	for _, result := range dnsResults {
		err := out.Execute(os.Stdout, result)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
			error = err
		}
	}
	return error
}

func showResultsHuman(dnsResults []dns.ExplodedDNS) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Domain", "Unique Subdomains", "Times Looked Up"})
	for _, result := range dnsResults {
		table.Append([]string{
			result.Domain,
			strconv.FormatInt(result.Subdomains, 10),
			strconv.FormatInt(result.Visited, 10),
		})
	}
	table.Render()
	return nil
}
