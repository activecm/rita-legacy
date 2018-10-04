package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/activecm/rita/resources"
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
			res := resources.InitResources(c.String("config"))
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}
			return resetAnalysis(db, res)
		},
	}

	bootstrapCommands(reset)
}

// resetAnalysis cleans out all of the analysis data, leaving behind only the
// raw data from parsing the logs
func resetAnalysis(database string, res *resources.Resources) error {
	//clean database

	conn := res.Config.T.Structure.ConnTable
	http := res.Config.T.Structure.HTTPTable
	dns := res.Config.T.Structure.DNSTable

	ritaDB, err := res.DBIndex.GetDatabase(database)
	if err != nil {
		return cli.NewExitError("Failed to find database "+database+".", -1)
	}

	/*
		Sometimes analysis fails and we need to reset a database which
		hasn't been properly marked analyzed.

		Alternatively, we create a flag for analysis start in the metadb

		if !ritaDB.Analyzed() {
			return cli.NewExitError("Database "+database+" has not been analyzed.", -1)
		}
	*/

	fmt.Print("Are you sure you want to reset analysis for ", database, " [y/N] ")

	read := bufio.NewReader(os.Stdin)

	response, err := read.ReadString('\n')
	if err != nil {
		return cli.NewExitError("Error: could not read response: "+err.Error(), -1)
	}
	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" || response == "yes" {
		fmt.Println("Resetting database:", database)
	} else {
		return cli.NewExitError("Database "+database+" was not reset.", 0)
	}

	names, err := res.DB.Session.DB(database).CollectionNames()
	if err != nil {
		return cli.NewExitError("Error: could not list analysis results: "+err.Error(), -1)
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
				fmt.Fprintf(os.Stderr, "Error: failed to drop collection: %s\n", err2.Error())
				err2Flag = err2
			}
		}
	}
	if err2Flag != nil {
		return cli.NewExitError("Error: could not remove some analysis results", -1)
	}

	//change metadb
	err = ritaDB.UnsetAnalyzed(res.DB.Session)
	if err != nil {
		return cli.NewExitError("Error: could not mark database "+database+" as not analyzed: "+err.Error(), -1)
	}

	fmt.Fprintf(os.Stdout, "Successfully reset analysis of %s.\n", database)
	return nil
}
