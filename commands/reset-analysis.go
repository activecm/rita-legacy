package commands

import (
	"fmt"
	"os"

	"github.com/ocmdev/rita/database"
	"github.com/urfave/cli"
)

func init() {
	reset := cli.Command{
		Name:  "reset-analysis",
		Usage: "Reset analysis of one or more databases",
		Flags: []cli.Flag{
			databaseFlag,
		},
		Action: func(c *cli.Context) error {
			res := database.InitResources("")
			if c.String("database") == "" {
				fmt.Println("Resetting all databases")
				return cleanAnalysisAll(res)
			}
			fmt.Println("Resetting database:", c.String("database"))
			return cleanAnalysis(c.String("database"), res)
		},
	}

	bootstrapCommands(reset)
}

// cleanAnalysis cleans out all of the analysis data, leaving behind only the
// raw data from parsing the logs
func cleanAnalysis(database string, res *database.Resources) error {
	//clean database

	conn := res.System.StructureConfig.ConnTable
	http := res.System.StructureConfig.HTTPTable
	dns := res.System.StructureConfig.DNSTable

	names, err := res.DB.Session.DB(database).CollectionNames()
	if err != nil || len(names) == 0 {
		fmt.Fprintf(os.Stderr, "Failed to find analysis results\n")
		return err
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
		fmt.Fprintf(os.Stderr, "Failed to update metadb\n")
		return err3
	}

	if err == nil && err2Flag == nil && err3 == nil {
		fmt.Fprintf(os.Stdout, "Successfully reset analysis of %s.\n", database)
	}
	return nil
}

// cleanAnalysisAll uses the metadb to walk all databases and clean the analysis
func cleanAnalysisAll(res *database.Resources) error {
	var err error

	for _, name := range res.MetaDB.GetDatabases() {
		e := cleanAnalysis(name, res)
		//return last error
		if e != nil {
			err = e
		}
	}
	return err
}
