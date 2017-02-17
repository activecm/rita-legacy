package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/mgo.v2/bson"

	"github.com/alecthomas/template"
	"github.com/ocmdev/rita/database"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

var sourcesFlag bool

type blresult struct {
	Host    string `bson:"host"`
	Score   int    `bson:"count"`
	IsUrl   bool   `bson:"is_url"`
	Sources []string
}

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
		},
		Action: showBlacklisted,
	}

	bootstrapCommands(command)
}

func showBlacklisted(c *cli.Context) error {
	if c.String("database") == "" {
		return cli.NewExitError("Specify a database with -d", -1)
	}

	res := database.InitResources("")
	res.DB.SelectDB(c.String("database"))

	var result blresult
	var results []blresult

	coll := res.DB.Session.DB(c.String("database")).C(res.System.BlacklistedConfig.BlacklistTable)
	iter := coll.Find(nil).Sort("-count").Iter()

	for iter.Next(&result) {
		if sourcesFlag {
			findBLSources(res, c.String("database"), &result)
		}
		results = append(results, result)
	}

	if humanreadable {
		return showBlacklistedHuman(results)
	}
	return showBlacklistedCsv(results)
}

func findBLSources(res *database.Resources, db string, result *blresult) {
	if result.IsUrl {
		hostnames := res.DB.Session.DB(db).C(res.System.UrlsConfig.HostnamesTable)
		var destIPs struct {
			IPs []string `bson:"ips"`
		}
		hostnames.Find(bson.M{"host": result.Host}).One(&destIPs)
		for _, destIP := range destIPs.IPs {
			result.Sources = append(result.Sources, getConnSourceFromDest(res, db, destIP)...)
		}
	} else {
		result.Sources = getConnSourceFromDest(res, db, result.Host)
	}
}

func getConnSourceFromDest(res *database.Resources, db string, ip string) []string {
	cons := res.DB.Session.DB(db).C(res.System.StructureConfig.UniqueConnTable)
	srcIter := cons.Find(bson.M{"dst": ip}).Iter()

	var srcStruct struct {
		Src string `bson:"src"`
	}
	var sources []string

	for srcIter.Next(&srcStruct) {
		sources = append(sources, srcStruct.Src)
	}
	return sources
}

// TODO: Convert this over to tablewriter
// showBlacklisted prints all blacklisted for a given database
func showBlacklistedHuman(results []blresult) error {
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

func showBlacklistedCsv(results []blresult) error {
	tmpl := "{{.Host}}," + `{{.Score}}`
	if sourcesFlag {
		tmpl += ",{{range $idx, $src := .Sources}}{{if $idx}} {{end}}{{ $src }}{{end}}\n"
	} else {
		tmpl += "\n"
	}
	out, err := template.New("bl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	for _, result := range results {
		err := out.Execute(os.Stdout, result)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}

	return nil
}
