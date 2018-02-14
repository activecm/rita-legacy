package commands

import (
	"encoding/csv"
	"os"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/dns"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:      "show-exploded-dns",
		Usage:     "Print dns analysis. Exposes covert dns channels",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
			configFlag,
		},
		Action: func(c *cli.Context) error {
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}

			res := database.InitResources(c.String("config"))

			var explodedResults []dns.ExplodedDNS
			iter := res.DB.Session.DB(db).C(res.Config.T.DNS.ExplodedDNSTable).Find(nil)

			iter.Sort("-subdomains").All(&explodedResults)

			if len(explodedResults) == 0 {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showDNSResultsHuman(explodedResults)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
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
	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Write([]string{"Domain", "Unique Subdomains", "Times Looked Up"})
	for _, result := range dnsResults {
		csvWriter.Write([]string{
			result.Domain, i(result.Subdomains), i(result.Visited),
		})
	}
	csvWriter.Flush()
	return nil
}

func showDNSResultsHuman(dnsResults []dns.ExplodedDNS) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Domain", "Unique Subdomains", "Times Looked Up"})
	for _, result := range dnsResults {
		table.Append([]string{
			result.Domain, i(result.Subdomains), i(result.Visited),
		})
	}
	table.Render()
	return nil
}
