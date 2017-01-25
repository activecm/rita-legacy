package commands

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/mgo.v2/bson"

	"github.com/alecthomas/template"
	"github.com/ocmdev/rita/database"
	"github.com/urfave/cli"
)

type blresult struct {
	Host    string `bson:"host"`
	Score   int    `bson:"count"`
	Sources string
}

var globalSourcesFlag bool

type blresults []blresult

func (slice blresults) Len() int {
	return len(slice)
}

func (slice blresults) Less(i, j int) bool {
	return slice[j].Score < slice[i].Score
}

func (slice blresults) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
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
				Destination: &globalSourcesFlag,
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

	if humanreadable {
		return showBlacklistedHuman(c)
	}

	tmpl := "{{.Host}}," + `{{.Score}}`
	if globalSourcesFlag {
		tmpl += ", {{.Sources}}\n"
	} else {
		tmpl += "\n"
	}
	out, err := template.New("bl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	res := database.InitResources("")
	res.DB.SelectDB(c.String("database"))

	var result blresult
	var allResults blresults

	coll := res.DB.Session.DB(c.String("database")).C(res.System.BlacklistedConfig.BlacklistTable)
	iter := coll.Find(nil).Iter()

	for iter.Next(&result) {
		if globalSourcesFlag {
			result.Sources = ""
			cons := res.DB.Session.DB(c.String("database")).C(res.System.StructureConfig.ConnTable)
			siter := cons.Find(bson.M{"id_resp_h": result.Host}).Iter()

			var srcStruct struct {
				Src string `bson:"id_origin_h"`
			}

			for siter.Next(&srcStruct) {
				result.Sources += srcStruct.Src + "; "
			}
		}
		allResults = append(allResults, result)
	}

	sort.Sort(allResults)

	for _, result := range allResults {
		err := out.Execute(os.Stdout, result)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}
	return nil
}

// TODO: Convert this over to tablewriter
// showBlacklisted prints all blacklisted for a given database
func showBlacklistedHuman(c *cli.Context) error {

	cols := "            host\tscore\n"
	cols += "----------------\t-----\n"
	tmpl := "{{.Host}}\t" + `{{.Score | printf "%5d"}}` + "\n"
	out, err := template.New("bl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	res := database.InitResources("")
	res.DB.SelectDB(c.String("database"))

	var result blresult
	var allResults blresults

	coll := res.DB.Session.DB(c.String("database")).C(res.System.BlacklistedConfig.BlacklistTable)
	iter := coll.Find(nil).Iter()

	fmt.Printf(cols)
	for iter.Next(&result) {
		result.Host = padAddr(result.Host)
		allResults = append(allResults, result)
	}

	sort.Sort(allResults)

	for _, result := range allResults {
		err := out.Execute(os.Stdout, result)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}
	return nil
}
