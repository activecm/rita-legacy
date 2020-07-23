package commands

import (
	"fmt"
	"os"
	"strings"

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

			sortStr := "conn_count"
			sortDirection := -1
			if c.Bool("connection-count") == false {
				sortDirection = 1
			}

			data, err := getStrobeResultsView(res, sortStr, sortDirection, c.Int("limit"), c.Bool("no-limit"))

			if err != nil {
				res.Log.Error(err)
				return cli.NewExitError(err, -1)
			}

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
			err = showStrobes(data, c.String("delimiter"))
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showStrobes(strobes []beacon.StrobeAnalysisView, delim string) error {
	headers := []string{"Source", "Destination", "Connection Count"}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headers, delim))
	for _, strobe := range strobes {
		fmt.Println(
			strings.Join(
				[]string{strobe.Src, strobe.Dst, i(strobe.ConnectionCount)},
				delim,
			),
		)
	}
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
func getStrobeResultsView(res *resources.Resources, sort string, sortDir, limit int, noLimit bool) ([]beacon.StrobeAnalysisView, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var strobes []beacon.StrobeAnalysisView

	strobeQuery := []bson.M{
		bson.M{"$match": bson.M{"strobe": true}},
		bson.M{"$unwind": "$dat"},
		bson.M{"$project": bson.M{"src": 1, "dst": 1, "conns": "$dat.count"}},
		bson.M{"$group": bson.M{
			"_id":        "$_id",
			"src":        bson.M{"$first": "$src"},
			"dst":        bson.M{"$first": "$dst"},
			"conn_count": bson.M{"$sum": "$conns"},
		}},
		bson.M{"$sort": bson.M{sort: sortDir}},
	}

	if !noLimit {
		strobeQuery = append(strobeQuery, bson.M{"$limit": limit})
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).Pipe(strobeQuery).AllowDiskUse().All(&strobes)

	return strobes, err

}
