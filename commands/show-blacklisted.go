package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/template"
	"github.com/ocmdev/rita/analysis/blacklisted"
	"github.com/ocmdev/rita/database"
	blacklistedData "github.com/ocmdev/rita/datatypes/blacklisted"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

var sourcesFlag bool

func init() {
	command := cli.Command{
		Name:  "show-blacklisted",
		Usage: "Print blacklisted information to standard out",
		Flags: []cli.Flag{
			databaseFlag,
			humanFlag,
			cli.BoolFlag{
				Name:        "sources, s",
				Usage:       "Show sources with results",
				Destination: &sourcesFlag,
			},
			configFlag,
		},
		Action: showBlacklisted,
	}

	bootstrapCommands(command)
}

func showBlacklisted(c *cli.Context) error {
	if c.String("database") == "" {
		return cli.NewExitError("Specify a database with -d", -1)
	}

	res := database.InitResources(c.String("config"))
	res.DB.SelectDB(c.String("database"))

	var result blacklistedData.Blacklist
	var results []blacklistedData.Blacklist

	coll := res.DB.Session.DB(c.String("database")).C(res.System.BlacklistedConfig.BlacklistTable)
	iter := coll.Find(nil).Sort("-count").Iter()

	for iter.Next(&result) {
		if sourcesFlag {
			blacklisted.SetBlacklistSources(res, &result)
		}
		results = append(results, result)
	}

	if c.Bool("human-readable") {
		return showBlacklistedHuman(results)
	}
	return showBlacklistedCsv(results)
}

// showBlacklisted prints all blacklisted for a given database
func showBlacklistedHuman(results []blacklistedData.Blacklist) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetColWidth(100)
	if sourcesFlag {
		table.SetHeader([]string{"Host", "Score", "Sources"})
		for _, result := range results {
			table.Append([]string{
				result.Host, strconv.Itoa(result.Score), strings.Join(result.Sources, ", "),
			})
		}
	} else {
		table.SetHeader([]string{"Host", "Score"})
		for _, result := range results {
			table.Append([]string{result.Host, strconv.Itoa(result.Score)})
		}
	}

	table.Render()
	return nil
}

func showBlacklistedCsv(results []blacklistedData.Blacklist) error {
	tmpl := "{{.Host}}," + `{{.Score}}`
	if sourcesFlag {
		tmpl += ",{{range $idx, $src := .Sources}}{{if $idx}} {{end}}{{ $src }}{{end}}\n"
	} else {
		tmpl += "\n"
	}
	out, err := template.New("bl").Parse(tmpl)
	if err != nil {
		return err
	}

	for _, result := range results {
		err := out.Execute(os.Stdout, result)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}

	return nil
}
