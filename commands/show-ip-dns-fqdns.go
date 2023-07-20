package commands

import (
	"fmt"
	"strings"

	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/resources"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:      "show-ip-dns-fqdns",
		Usage:     "Print FQDNs associated with IP via DNS",
		ArgsUsage: "<database> <ip address>",
		Flags: []cli.Flag{
			ConfigFlag,
			humanFlag,
			delimFlag,
		},
		Action: showIpFqdns,
	}

	bootstrapCommands(command)
}

func showIpFqdns(c *cli.Context) error {
	db := c.Args().Get(0)
	if db == "" {
		return cli.NewExitError("Specify a database", -1)
	}

	ip := c.Args().Get(1)
	if ip == "" {
		return cli.NewExitError("Specify an IP address", -1)
	}
	res := resources.InitResources(getConfigFilePath(c))
	res.DB.SelectDB(db)

	fqdnResults, err := hostname.FQDNResults(res, ip)

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if !(len(fqdnResults) > 0) {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	err = showIpFqdnsDelim(fqdnResults, c.String("delimiter"))
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}

	return nil
}

func showIpFqdnsDelim(data []*hostname.FQDNResult, delim string) error {
	headerFields := []string{
		"Resolved FQDNs",
	}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headerFields, delim))
	for _, d := range data {
		row := []string{
			d.Hostname,
		}

		fmt.Println(strings.Join(row, delim))
	}
	return nil
}
