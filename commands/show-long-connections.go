package commands

import (
	"encoding/csv"
	"os"
	"strings"

	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:      "show-long-connections",
		Usage:     "Print long connections and relevant information",
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

			sortStr := "maxdur"
			sortDirection := -1
			thresh := 60 // 1 minute

			data, err := getLongConnsResultsView(res, thresh, sortStr, sortDirection, c.Int("limit"))

			if err != nil {
				res.Log.Error(err)
				return cli.NewExitError(err, -1)
			}

			if !(len(data) > 0) {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showConnsHuman(data)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err = showConns(data)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showConns(connResults []uconn.LongConnAnalysisView) error {
	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Write([]string{"Source IP", "Destination IP",
		"Port:Protocol:Service", "Duration"})
	for _, result := range connResults {
		csvWriter.Write([]string{
			result.Src,
			result.Dst,
			strings.Join(result.Tuples, " "),
			f(result.MaxDuration),
		})
	}
	csvWriter.Flush()
	return nil
}

func showConnsHuman(connResults []uconn.LongConnAnalysisView) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Source IP", "Destination IP",
		"DstPort:Protocol:Service", "Duration"})
	for _, result := range connResults {
		table.Append([]string{
			result.Src,
			result.Dst,
			strings.Join(result.Tuples, ",\n"),
			f(result.MaxDuration) + "s",
		})
	}
	table.Render()
	return nil
}

//getLongConnsResultsView gets the long connection results
func getLongConnsResultsView(res *resources.Resources, thresh int, sort string, sortDirection int, limit int) ([]uconn.LongConnAnalysisView, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var longConnResults []uconn.LongConnAnalysisView

	longConnQuery := []bson.M{
		bson.M{"$match": bson.M{"dat.maxdur": bson.M{"$gt": thresh}}},
		bson.M{"$project": bson.M{"maxdur": "$dat.maxdur", "src": "$src", "dst": "$dst", "tuples": bson.M{"$ifNull": []interface{}{"$dat.tuples", []interface{}{}}}}},
		bson.M{"$unwind": "$maxdur"},
		bson.M{"$unwind": "$tuples"},
		bson.M{"$unwind": "$tuples"}, // not an error, must be done twice
		bson.M{"$group": bson.M{
			"_id":    "$_id",
			"maxdur": bson.M{"$max": "$maxdur"},
			"src":    bson.M{"$first": "$src"},
			"dst":    bson.M{"$first": "$dst"},
			"tuples": bson.M{"$addToSet": "$tuples"},
		}},
		bson.M{"$project": bson.M{
			"maxdur": 1,
			"src":    1,
			"dst":    1,
			"tuples": bson.M{"$slice": []interface{}{"$tuples", 5}},
		}},
		bson.M{"$sort": bson.M{sort: sortDirection}},
		bson.M{"$limit": limit},
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).Pipe(longConnQuery).AllowDiskUse().All(&longConnResults)

	return longConnResults, err

}
