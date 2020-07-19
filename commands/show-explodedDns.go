package commands

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/activecm/rita/pkg/explodeddns"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
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
			limitFlag,
			noLimitFlag,
			delimFlag,
		},
		Action: func(c *cli.Context) error {
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}

			res := resources.InitResources(c.String("config"))
			res.DB.SelectDB(db)

			data, err := getExplodedDNSResultsView(res, c.Int("limit"), c.Bool("no-limit"))

			if err != nil {
				res.Log.Error(err)
				return cli.NewExitError(err, -1)
			}

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
			err = showDNSResults(data, c.String("delimiter"))
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

// splitSubN splits s every n characters
func splitSubN(s string, n int) []string {
	sub := ""
	subs := []string{}

	runes := bytes.Runes([]byte(s))
	l := len(runes)
	for i, r := range runes {
		sub = sub + string(r)
		if (i+1)%n == 0 {
			subs = append(subs, sub)
			sub = ""
		} else if (i + 1) == l {
			subs = append(subs, sub)
		}
	}

	return subs
}

func showDNSResults(dnsResults []explodeddns.AnalysisView, delim string) error {
	headers := []string{"Domain", "Unique Subdomains", "Times Looked Up"}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headers, delim))
	for _, result := range dnsResults {
		fmt.Println(
			strings.Join(
				[]string{result.Domain, i(result.SubdomainCount), i(result.Visited)},
				delim,
			),
		)
	}
	return nil
}

func showDNSResultsHuman(dnsResults []explodeddns.AnalysisView) error {
	const DOMAINRECLEN = 80
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(true)
	table.SetRowSeparator("-")
	table.SetRowLine(true)
	table.SetHeader([]string{"Domain", "Unique Subdomains", "Times Looked Up"})
	for _, result := range dnsResults {
		domain := result.Domain
		if len(domain) > DOMAINRECLEN {
			// Reformat the result.Domain value adding a newline every DOMAINRECLEN chars for wrapping
			subs := splitSubN(result.Domain, DOMAINRECLEN)
			domain = strings.Join(subs, "\n")
		}
		table.Append([]string{
			domain, i(result.SubdomainCount), i(result.Visited),
		})
	}
	table.Render()
	return nil
}

//getExplodedDNSResultsView gets the exploded dns results
func getExplodedDNSResultsView(res *resources.Resources, limit int, noLimit bool) ([]explodeddns.AnalysisView, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var explodedDNSResults []explodeddns.AnalysisView

	explodedDNSQuery := []bson.M{
		bson.M{"$unwind": "$dat"},
		bson.M{"$project": bson.M{"domain": 1, "subdomain_count": 1, "visited": "$dat.visited"}},
		bson.M{"$group": bson.M{
			"_id":             "$domain",
			"visited":         bson.M{"$sum": "$visited"},
			"subdomain_count": bson.M{"$first": "$subdomain_count"},
		}},
		bson.M{"$project": bson.M{
			"_id":             0,
			"domain":          "$_id",
			"visited":         1,
			"subdomain_count": 1,
		}},
		bson.M{"$sort": bson.M{"visited": -1}},
		bson.M{"$sort": bson.M{"subdomain_count": -1}},
	}

	if !noLimit {
		explodedDNSQuery = append(explodedDNSQuery, bson.M{"$limit": limit})
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.ExplodedDNSTable).Pipe(explodedDNSQuery).AllowDiskUse().All(&explodedDNSResults)

	return explodedDNSResults, err

}
