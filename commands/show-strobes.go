package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/activecm/rita/pkg/beacon"
	"github.com/activecm/rita/resources"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:      "show-strobes",
		Usage:     "Print strobe information",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			ConfigFlag,
			humanFlag,
			cli.BoolFlag{
				Name:  "connection-count, l",
				Usage: "Sort the strobes by largest connection count.",
			},
			limitFlag,
			noLimitFlag,
			delimFlag,
			netNamesFlag,
		},
		Action: func(c *cli.Context) error {
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}

			res := resources.InitResources(getConfigFilePath(c))
			res.DB.SelectDB(db)

			sortDirection := -1
			if !c.Bool("connection-count") {
				sortDirection = 1
			}

			data, err := beacon.StrobeResults(res, sortDirection, c.Int("limit"), c.Bool("no-limit"))

			if err != nil {
				res.Log.Error(err)
				return cli.NewExitError(err, -1)
			}

			if len(data) == 0 {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showStrobesHuman(data, c.Bool("network-names"))
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err = showStrobes(data, c.String("delimiter"), c.Bool("network-names"))
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showStrobes(strobes []beacon.StrobeResult, delim string, showNetNames bool) error {
	var headerFields []string
	if showNetNames {
		headerFields = []string{"Source Network", "Destination Network", "Source", "Destination", "Connection Count"}
	} else {
		headerFields = []string{"Source", "Destination", "Connection Count"}
	}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headerFields, delim))
	for _, strobe := range strobes {
		var row []string
		if showNetNames {
			row = []string{
				strobe.SrcNetworkName,
				strobe.DstNetworkName,
				strobe.SrcIP,
				strobe.DstIP,
				i(strobe.ConnectionCount),
			}
		} else {
			row = []string{
				strobe.SrcIP,
				strobe.DstIP,
				i(strobe.ConnectionCount),
			}
		}
		fmt.Println(strings.Join(row, delim))
	}
	return nil
}

func showStrobesHuman(strobes []beacon.StrobeResult, showNetNames bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetColWidth(100)

	var headerFields []string
	if showNetNames {
		headerFields = []string{"Source Network", "Destination Network", "Source", "Destination", "Connection Count"}
	} else {
		headerFields = []string{"Source", "Destination", "Connection Count"}
	}
	table.SetHeader(headerFields)

	for _, strobe := range strobes {
		var row []string
		if showNetNames {
			row = []string{
				strobe.SrcNetworkName,
				strobe.DstNetworkName,
				strobe.SrcIP,
				strobe.DstIP,
				i(strobe.ConnectionCount),
			}
		} else {
			row = []string{
				strobe.SrcIP,
				strobe.DstIP,
				i(strobe.ConnectionCount),
			}
		}
		table.Append(row)
	}
	table.Render()
	return nil
}
