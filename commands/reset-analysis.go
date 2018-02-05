package commands

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ocmdev/rita/database"
	"github.com/urfave/cli"
)

func init() {
	reset := cli.Command{
		Name:      "reset-analysis",
		Usage:     "Reset analysis of a database",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			configFlag,
		},
		Action: func(c *cli.Context) error {
			res := database.InitResources(c.String("config"))
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}
			return cleanAnalysis(db, res)
		},
	}

	bootstrapCommands(reset)
}

// cleanAnalysis cleans out all of the analysis data, leaving behind only the
// raw data from parsing the logs
func cleanAnalysis(database string, res *database.Resources) error {
	//clean database

	conn := res.Config.T.Structure.ConnTable
	http := res.Config.T.Structure.HTTPTable
	dns := res.Config.T.Structure.DNSTable

	names, err := res.DB.Session.DB(database).CollectionNames()
	if err != nil || len(names) == 0 {
		return cli.NewExitError("Failed to find analysis results", -1)
	}

	fmt.Println("Are you sure you want to reset analysis for", database, "[Y/n]")

	read := bufio.NewReader(os.Stdin)

	response, err := read.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" || response == "yes" {
		fmt.Println("Resetting database:", database)
	} else {
		fmt.Println("Aborted, nothing reset")
		return nil
	}

	//check if we had an issue dropping a collection
	var err2Flag error
	for _, name := range names {
		switch name {
		case conn, http, dns:
			continue
		default:
			err2 := res.DB.Session.DB(database).C(name).DropCollection()
			if err2 != nil {
				fmt.Fprintf(os.Stderr, "Failed to drop collection: %s\n", err2.Error())
				err2Flag = err2
			}
		}
	}

	//change metadb
	err3 := res.MetaDB.MarkDBAnalyzed(database, false)

	if err3 != nil {
		return cli.NewExitError("Failed to update metadb", -1)
	}

	if err == nil && err2Flag == nil && err3 == nil {
		fmt.Fprintf(os.Stdout, "Successfully reset analysis of %s.\n", database)
	}
	return nil
}
