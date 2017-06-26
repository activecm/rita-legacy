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
			configFlag,
		},
		Action: func(c *cli.Context) error {
			if c.String("database") == "" {
				return cli.NewExitError("Specify a database with -d", -1)
			}

			res := database.InitResources(c.String("config"))

			var explodedResults []dns.ExplodedDNS
			iter := res.DB.Session.DB(c.String("database")).C(res.System.DNSConfig.ExplodedDNSTable).Find(nil)

			iter.Sort("-subdomains").All(&explodedResults)

			if len(explodedResults) == 0 {
				return cli.NewExitError("No results were found for "+c.String("database"), -1)
			}

			if c.Bool("human-readable") {
				err := showDNSResultsHuman(explodedResults)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
			}
			err := showDNSResults(explodedResults)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showDNSResults(dnsResults []dns.ExplodedDNS) error {
	tmpl := "{{.Domain}},{{.Subdomains}},{{.Visited}}\n"

	out, err := template.New("exploded-dns").Parse(tmpl)
	if err != nil {
		return err
	}

	for _, result := range dnsResults {
		err := out.Execute(os.Stdout, result)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}
	return nil
}

func showDNSResultsHuman(dnsResults []dns.ExplodedDNS) error {
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
