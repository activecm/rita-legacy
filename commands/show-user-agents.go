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
			allFlag,
			cli.BoolFlag{
				Name:  "least-used, l",
				Usage: "Print the least used user agent strings",
			},
		},
		Action: func(c *cli.Context) error {
			if c.String("database") == "" {
				return cli.NewExitError("Specify a database with -d", -1)
			}

			res := database.InitResources("")

			var agents []useragent.UserAgent
			coll := res.DB.Session.DB(c.String("database")).C(res.System.UserAgentConfig.UserAgentTable)

			var sortStr string
			if c.Bool("least-used") {
				sortStr = "times_used"
			} else {
				sortStr = "-times_used"
			}

			query := coll.Find(nil).Sort(sortStr)
			if !c.Bool("all") {
				query.Limit(15)
			}
			query.All(&agents)

			if c.Bool("human-readable") {
				return showAgentsHuman(agents)
			}
			return showAgents(agents)
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

	var error error
	for _, agent := range agents {
		err := out.Execute(os.Stdout, agent)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
			error = err
		}
	}
	return error
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
