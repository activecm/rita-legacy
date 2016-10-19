package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/alecthomas/template"
	"github.com/ocmdev/rita/config"
	"github.com/urfave/cli"
)

type blresult struct {
	Host  string `bson:"host"`
	Score int    `bson:"count"`
}

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
		Usage: "print blacklisted information to standard out",
		Flags: []cli.Flag{
			databaseFlag,
			humanFlag,
		},
		Action: showBlacklisted,
	}

	bootstrapCommands(command)
}

func showBlacklisted(c *cli.Context) error {

	if humanreadable {
		return showBlacklistedHuman(c)
	}

	tmpl := "{{.Host}}," + `{{.Score}}` + "\n"
	out, err := template.New("bl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")
	conf.System.DB = c.String("dataset")

	var res blresult
	var allres blresults

	coll := conf.Session.DB(c.String("dataset")).C(conf.System.BlacklistedConfig.BlacklistTable)
	iter := coll.Find(nil).Iter()

	for iter.Next(&res) {
		allres = append(allres, res)
	}

	sort.Sort(allres)

	for _, res := range allres {
		err := out.Execute(os.Stdout, res)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}
	return nil
}

// showBlacklisted prints all blacklisted for a given database
func showBlacklistedHuman(c *cli.Context) error {

	cols := "            host\tscore\n"
	cols += "----------------\t-----\n"
	tmpl := "{{.Host}}\t" + `{{.Score | printf "%5d"}}` + "\n"
	out, err := template.New("bl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")
	conf.System.DB = c.String("dataset")

	var res blresult
	var allres blresults

	coll := conf.Session.DB(c.String("dataset")).C(conf.System.BlacklistedConfig.BlacklistTable)
	iter := coll.Find(nil).Iter()

	fmt.Printf(cols)
	for iter.Next(&res) {
		res.Host = padAddr(res.Host)
		allres = append(allres, res)
	}

	sort.Sort(allres)

	for _, res := range allres {
		err := out.Execute(os.Stdout, res)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}
	return nil
}
