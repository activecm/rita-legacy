package commands

import (
	"encoding/csv"
	"os"
	"strconv"

	"github.com/activecm/rita/pkg/conn"
	"github.com/activecm/rita/resources"
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

			var longConns []conn.AnalysisView
			coll := res.DB.Session.DB(db).C(res.Config.T.Structure.ConnTable)

			sortStr := "-duration"

			coll.Find(nil).Sort(sortStr).All(&longConns)

			if len(longConns) == 0 {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showConnsHuman(longConns)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err := showConns(longConns)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showConns(connResults []conn.AnalysisView) error {
	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Write([]string{"Source IP", "Source Port", "Destination IP",
		"Destination Port", "Duration", "Protocol"})
	for _, result := range connResults {
		csvWriter.Write([]string{
			result.Src,
			strconv.Itoa(result.Spt),
			result.Dst,
			strconv.Itoa(result.Dpt),
			f(result.Dur),
			result.Proto,
		})
	}
	csvWriter.Flush()
	return nil
}

func showConnsHuman(connResults []conn.AnalysisView) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Source IP", "Source Port", "Destination IP",
		"Destination Port", "Duration", "Protocol"})
	for _, result := range connResults {
		table.Append([]string{
			result.Src,
			strconv.Itoa(result.Spt),
			result.Dst,
			strconv.Itoa(result.Dpt),
			f(result.Dur),
			result.Proto,
		})
	}
	table.Render()
	return nil
}
