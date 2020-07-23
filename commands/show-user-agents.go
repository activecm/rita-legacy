package commands

import (
	"fmt"
	"os"
	"strings"

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

			sort := "seen"
			sortDirection := 1
			if c.Bool("least-used") == false {
				sortDirection = -1
			}

			data, err := getUseragentResultsView(res, sort, sortDirection, c.Int("limit"), c.Bool("no-limit"))

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
			err = showAgents(data, c.String("delimiter"))
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showAgents(agents []useragent.AnalysisView, delim string) error {
	headers := []string{"User Agent", "Times Used"}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headers, delim))
	for _, agent := range agents {
		fmt.Println(
			strings.Join(
				[]string{agent.UserAgent, i(agent.TimesUsed)},
				delim,
			),
		)
	}
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
func getUseragentResultsView(res *resources.Resources, sort string, sortDirection, limit int, noLimit bool) ([]useragent.AnalysisView, error) {
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
	}

	if !noLimit {
		useragentQuery = append(useragentQuery, bson.M{"$limit": limit})
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.UserAgent.UserAgentTable).Pipe(useragentQuery).AllowDiskUse().All(&useragentResults)

	return useragentResults, err

}
