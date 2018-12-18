package commands

import (
	"encoding/csv"
	"os"

	"github.com/activecm/rita/datatypes/strobe"
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
			humanFlag,
			cli.BoolFlag{
				Name:  "connection-count, l",
				Usage: "Sort the strobes by largest connection count.",
			},
			configFlag,
		},
		Action: func(c *cli.Context) error {
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}

			res := resources.InitResources(c.String("config"))

			var strobes []strobe.Strobe
			coll := res.DB.Session.DB(db).C(res.Config.T.Strobe.StrobeTable)

			var sortStr string
			if c.Bool("connection-count") {
				sortStr = "connection_count"
			} else {
				sortStr = "-connection_count"
			}

			coll.Find(nil).Sort(sortStr).All(&strobes)

			if len(strobes) == 0 {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("pretty") {
				err := showStrobesHuman(strobes)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err := showStrobes(strobes)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showStrobes(strobes []strobe.Strobe) error {
	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Write([]string{"Source", "Destination", "Connection Count"})
	for _, strobe := range strobes {
		csvWriter.Write([]string{strobe.Source, strobe.Destination, i(strobe.ConnectionCount)})
	}
	csvWriter.Flush()
	return nil
}

func showStrobesHuman(strobes []strobe.Strobe) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetColWidth(100)
	table.SetHeader([]string{"Source", "Destination", "Connection Count"})
	for _, strobe := range strobes {
		table.Append([]string{strobe.Source, strobe.Destination, i(strobe.ConnectionCount)})
	}
	table.Render()
	return nil
}
