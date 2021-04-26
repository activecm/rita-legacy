package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/activecm/rita/pkg/beaconfqdn"
	"github.com/activecm/rita/resources"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:      "show-beacons-fqdn",
		Usage:     "Print hosts which show signs of C2 software (FQDN Analysis)",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			configFlag,
			humanFlag,
			delimFlag,
			netNamesFlag,
		},
		Action: showBeaconsFQDN,
	}

	bootstrapCommands(command)
}

func showBeaconsFQDN(c *cli.Context) error {
	db := c.Args().Get(0)
	if db == "" {
		return cli.NewExitError("Specify a database", -1)
	}
	res := resources.InitResources(getConfigFilePath(c))
	res.DB.SelectDB(db)

	data, err := beaconfqdn.Results(res, 0)

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if !(len(data) > 0) {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	showNetNames := c.Bool("network-names")

	if c.Bool("human-readable") {
		err := showBeaconsFQDNHuman(data, showNetNames)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
		return nil
	}

	err = showBeaconsFQDNDelim(data, c.String("delimiter"), showNetNames)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}
	return nil
}

func showBeaconsFQDNHuman(data []beaconfqdn.Result, showNetNames bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	var headerFields []string
	if showNetNames {
		headerFields = []string{
			"Score", "Source Network", "Source IP", "FQDN",
			"Connections", "Avg. Bytes", "Intvl Range", "Size Range", "Top Intvl",
			"Top Size", "Top Intvl Count", "Top Size Count", "Intvl Skew",
			"Size Skew", "Intvl Dispersion", "Size Dispersion",
		}
	} else {
		headerFields = []string{
			"Score", "Source IP", "FQDN",
			"Connections", "Avg. Bytes", "Intvl Range", "Size Range", "Top Intvl",
			"Top Size", "Top Intvl Count", "Top Size Count", "Intvl Skew",
			"Size Skew", "Intvl Dispersion", "Size Dispersion",
		}
	}

	table.SetHeader(headerFields)

	for _, d := range data {
		var row []string

		if showNetNames {
			row = []string{
				f(d.Score), d.SrcNetworkName,
				d.SrcIP, d.FQDN, i(d.Connections), f(d.AvgBytes),
				i(d.Ts.Range), i(d.Ds.Range), i(d.Ts.Mode), i(d.Ds.Mode),
				i(d.Ts.ModeCount), i(d.Ds.ModeCount), f(d.Ts.Skew), f(d.Ds.Skew),
				i(d.Ts.Dispersion), i(d.Ds.Dispersion),
			}
		} else {
			row = []string{
				f(d.Score), d.SrcIP, d.FQDN, i(d.Connections), f(d.AvgBytes),
				i(d.Ts.Range), i(d.Ds.Range), i(d.Ts.Mode), i(d.Ds.Mode),
				i(d.Ts.ModeCount), i(d.Ds.ModeCount), f(d.Ts.Skew), f(d.Ds.Skew),
				i(d.Ts.Dispersion), i(d.Ds.Dispersion),
			}
		}
		table.Append(row)
	}
	table.Render()
	return nil
}

func showBeaconsFQDNDelim(data []beaconfqdn.Result, delim string, showNetNames bool) error {
	var headerFields []string
	if showNetNames {
		headerFields = []string{
			"Score", "Source Network", "Source IP", "FQDN",
			"Connections", "Avg. Bytes", "Intvl Range", "Size Range", "Top Intvl",
			"Top Size", "Top Intvl Count", "Top Size Count", "Intvl Skew",
			"Size Skew", "Intvl Dispersion", "Size Dispersion",
		}
	} else {
		headerFields = []string{
			"Score", "Source IP", "FQDN",
			"Connections", "Avg. Bytes", "Intvl Range", "Size Range", "Top Intvl",
			"Top Size", "Top Intvl Count", "Top Size Count", "Intvl Skew",
			"Size Skew", "Intvl Dispersion", "Size Dispersion",
		}
	}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headerFields, delim))
	for _, d := range data {

		var row []string
		if showNetNames {
			row = []string{
				f(d.Score), d.SrcNetworkName,
				d.SrcIP, d.FQDN, i(d.Connections), f(d.AvgBytes),
				i(d.Ts.Range), i(d.Ds.Range), i(d.Ts.Mode), i(d.Ds.Mode),
				i(d.Ts.ModeCount), i(d.Ds.ModeCount), f(d.Ts.Skew), f(d.Ds.Skew),
				i(d.Ts.Dispersion), i(d.Ds.Dispersion),
			}
		} else {
			row = []string{
				f(d.Score), d.SrcIP, d.FQDN, i(d.Connections), f(d.AvgBytes),
				i(d.Ts.Range), i(d.Ds.Range), i(d.Ts.Mode), i(d.Ds.Mode),
				i(d.Ts.ModeCount), i(d.Ds.ModeCount), f(d.Ts.Skew), f(d.Ds.Skew),
				i(d.Ts.Dispersion), i(d.Ds.Dispersion),
			}
		}

		fmt.Println(strings.Join(row, delim))
	}
	return nil
}
