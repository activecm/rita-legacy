package commands

import (
	"encoding/csv"
	"os"

	"github.com/activecm/rita/pkg/explodeddns"
	"github.com/activecm/rita/resources"
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

			res := resources.InitResources(c.String("config"))
			res.DB.SelectDB(db)

			data := getExplodedDNSResultsView(res)

			if len(data) == 0 {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showDNSResultsHuman(data)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err := showDNSResults(data)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

//getExplodedDNSResultsView gets the exploded dns results
func getExplodedDNSResultsView(res *resources.Resources) []explodeddns.AnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var explodedDNSResults []explodeddns.AnalysisView

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.ExplodedDNSTable).Find(nil).Sort("-subdomain_count").All(&explodedDNSResults)

	if err != nil {
		cli.NewExitError(err.Error(), -1)
	}

	return explodedDNSResults

}

func showDNSResults(dnsResults []explodeddns.AnalysisView) error {
	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Write([]string{"Domain", "Unique Subdomains", "Times Looked Up"})
	for _, result := range dnsResults {
		csvWriter.Write([]string{
			result.Domain, i(result.SubdomainCount), i(result.Visited),
		})
	}
	csvWriter.Flush()
	return nil
}

func showDNSResultsHuman(dnsResults []explodeddns.AnalysisView) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Domain", "Unique Subdomains", "Times Looked Up"})
	for _, result := range dnsResults {
		table.Append([]string{
			result.Domain, i(result.SubdomainCount), i(result.Visited),
		})
	}
	table.Render()
	return nil
}
