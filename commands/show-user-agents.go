package commands

import (
	"fmt"
	"os"
	"strconv"
	"text/template"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/useragent"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:  "show-user-agents",
		Usage: "Print user agent information",
		Flags: []cli.Flag{
			humanFlag,
			databaseFlag,
			cli.BoolFlag{
				Name:  "least-used, l",
				Usage: "Sort the user agents from least used to most used.",
			},
			configFlag,
		},
		Action: func(c *cli.Context) error {
			if c.String("database") == "" {
				return cli.NewExitError("Specify a database with -d", -1)
			}

			res := database.InitResources(c.String("config"))

			var agents []useragent.UserAgent
			coll := res.DB.Session.DB(c.String("database")).C(res.System.UserAgentConfig.UserAgentTable)

			var sortStr string
			if c.Bool("least-used") {
				sortStr = "times_used"
			} else {
				sortStr = "-times_used"
			}

			coll.Find(nil).Sort(sortStr).All(&agents)

			if len(agents) == 0 {
				return cli.NewExitError("No results were found for "+c.String("database"), -1)
			}

			if c.Bool("human-readable") {
				err := showAgentsHuman(agents)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
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
	tmpl := "{{.UserAgent}},{{.TimesUsed}}\n"

	out, err := template.New("ua").Parse(tmpl)
	if err != nil {
		return err
	}

	for _, agent := range agents {
		err := out.Execute(os.Stdout, agent)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}
	return nil
}

func showAgentsHuman(agents []useragent.UserAgent) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetColWidth(100)
	table.SetHeader([]string{"User Agent", "Times Used"})
	for _, agent := range agents {
		table.Append([]string{agent.UserAgent, strconv.FormatInt(agent.TimesUsed, 10)})
	}
	table.Render()
	return nil
}
