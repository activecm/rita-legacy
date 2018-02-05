package commands

import (
	"encoding/csv"
	"os"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/useragent"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:      "show-user-agents",
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

			res := database.InitResources(c.String("config"))

			var agents []useragent.UserAgent
			coll := res.DB.Session.DB(db).C(res.Config.T.UserAgent.UserAgentTable)

			var sortStr string
			if c.Bool("least-used") {
				sortStr = "times_used"
			} else {
				sortStr = "-times_used"
			}

			coll.Find(nil).Sort(sortStr).All(&agents)

			if len(agents) == 0 {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showAgentsHuman(agents)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err := showAgents(agents)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showAgents(agents []useragent.UserAgent) error {
	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Write([]string{"User Agent", "Times Used"})
	for _, agent := range agents {
		csvWriter.Write([]string{agent.UserAgent, i(agent.TimesUsed)})
	}
	csvWriter.Flush()
	return nil
}

func showAgentsHuman(agents []useragent.UserAgent) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetColWidth(100)
	table.SetHeader([]string{"User Agent", "Times Used"})
	for _, agent := range agents {
		table.Append([]string{agent.UserAgent, i(agent.TimesUsed)})
	}
	table.Render()
	return nil
}
