package commands

import (
	"fmt"
	"os"
	"strconv"
	"text/template"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/data"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:  "show-long-connections",
		Usage: "Print long connections and relevent information",
		Flags: []cli.Flag{
			humanFlag,
			databaseFlag,
			allFlag,
		},
		Action: func(c *cli.Context) error {
			if c.String("database") == "" {
				return cli.NewExitError("Specify a database with -d", -1)
			}

			res := database.InitResources("")

			var longConns []data.Conn
			coll := res.DB.Session.DB(c.String("database")).C(res.System.StructureConfig.ConnTable)

			sortStr := "-duration"

			query := coll.Find(nil).Sort(sortStr)
			if !c.Bool("all") {
				query.Limit(10)
			}
			query.All(&longConns)

			if c.Bool("human-readable") {
				return showConnsHuman(longConns)
			}
			return showConns(longConns)
		},
	}
	bootstrapCommands(command)
}

func showConns(connResults []data.Conn) error {
	tmpl := "{{.Src}},{{.Spt}},{{.Dst}},{{.Dpt}},{{.Dur}},{{.Proto}}\n"

	out, err := template.New("Conn").Parse(tmpl)
	if err != nil {
		return err
	}

	var error error
	for _, result := range connResults {
		err := out.Execute(os.Stdout, result)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
			error = err
		}
	}
	return error
}

func showConnsHuman(connResults []data.Conn) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Source IP", "Source Port", "Destination IP",
		"Destination Port", "Duration", "Protocol"})
	for _, result := range connResults {
		table.Append([]string{
			result.Src,
			strconv.Itoa(result.Spt),
			result.Dst,
			strconv.Itoa(result.Dpt),
			strconv.FormatFloat(result.Dur, 'f', 2, 64),
			result.Proto,
		})
	}
	table.Render()
	return nil
}
