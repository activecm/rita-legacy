package commands

import (
	"bytes"
	"fmt"
	"os"
	"strings"

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
			ConfigFlag,
			humanFlag,
			limitFlag,
			noLimitFlag,
			delimFlag,
		},
		Action: func(c *cli.Context) error {
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}

			res := resources.InitResources(getConfigFilePath(c))
			res.DB.SelectDB(db)

			data, err := explodeddns.Results(res, c.Int("limit"), c.Bool("no-limit"))

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

func showDNSResults(dnsResults []explodeddns.Result, delim string) error {
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

func showDNSResultsHuman(dnsResults []explodeddns.Result) error {
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
