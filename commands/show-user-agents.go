package commands

import (
	"encoding/csv"
	"os"

	"github.com/activecm/rita/pkg/useragent"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:      "show-useragents",
		Usage:     "Print user agent information",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
			cli.BoolFlag{
				Name:  "least-used, l",
				Usage: "Sort the user agents from least used to most used.",
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

			sort := "seen"
			sortDirection := 1
			if c.Bool("least-used") == false {
				sortDirection = -1
			}

			data, err := getUseragentResultsView(res, sort, sortDirection, 1000)

			if err != nil {
				res.Log.Error(err)
				return cli.NewExitError(err, -1)
			}

			if len(data) == 0 {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showAgentsHuman(data)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err = showAgents(data)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showAgents(agents []useragent.AnalysisView) error {
	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Write([]string{"User Agent", "Times Used"})
	for _, agent := range agents {
		csvWriter.Write([]string{agent.UserAgent, i(agent.TimesUsed)})
	}
	csvWriter.Flush()
	return nil
}

func showAgentsHuman(agents []useragent.AnalysisView) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetColWidth(100)
	table.SetHeader([]string{"User Agent", "Times Used"})
	for _, agent := range agents {
		table.Append([]string{agent.UserAgent, i(agent.TimesUsed)})
	}
	table.Render()
	return nil
}

//getUseragentResultsView gets the useragent results
func getUseragentResultsView(res *resources.Resources, sort string, sortDirection int, limit int) ([]useragent.AnalysisView, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var useragentResults []useragent.AnalysisView

	useragentQuery := []bson.M{
		bson.M{"$project": bson.M{"user_agent": 1, "seen": "$dat.seen"}},
		bson.M{"$unwind": "$seen"},
		bson.M{"$group": bson.M{
			"_id":  "$user_agent",
			"seen": bson.M{"$sum": "$seen"},
		}},
		bson.M{"$project": bson.M{
			"_id":        0,
			"user_agent": "$_id",
			"seen":       1,
		}},
		bson.M{"$sort": bson.M{sort: sortDirection}},
		bson.M{"$limit": limit},
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.UserAgent.UserAgentTable).Pipe(useragentQuery).All(&useragentResults)

	return useragentResults, err

}
