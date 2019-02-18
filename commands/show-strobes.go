package commands

import (
	"encoding/csv"
	"os"

	"github.com/activecm/rita/pkg/beacon"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
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
			res.DB.SelectDB(db)

			var sortStr string
			if c.Bool("connection-count") {
				sortStr = "connection_count"
			} else {
				sortStr = "-connection_count"
			}

			data := getStrobeResultsView(res, sortStr)

			if len(data) == 0 {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showStrobesHuman(data)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err := showStrobes(data)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showStrobes(strobes []beacon.StrobeAnalysisView) error {
	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Write([]string{"Source", "Destination", "Connection Count"})
	for _, strobe := range strobes {
		csvWriter.Write([]string{strobe.Src, strobe.Dst, i(strobe.ConnectionCount)})
	}
	csvWriter.Flush()
	return nil
}

func showStrobesHuman(strobes []beacon.StrobeAnalysisView) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetColWidth(100)
	table.SetHeader([]string{"Source", "Destination", "Connection Count"})
	for _, strobe := range strobes {
		table.Append([]string{strobe.Src, strobe.Dst, i(strobe.ConnectionCount)})
	}
	table.Render()
	return nil
}

//getStrobeResultsView ...
func getStrobeResultsView(res *resources.Resources, sort string) []beacon.StrobeAnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var strobes []beacon.StrobeAnalysisView

	strobeQuery := bson.M{"strobe": true}

	_ = ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).Find(strobeQuery).Sort(sort).All(&strobes)

	return strobes

}
