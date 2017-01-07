package commands

import (
	"fmt"
	"os"

	"github.com/ocmdev/rita/config"
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
			conf := config.InitConfig("")
			dbm := database.NewMetaDBHandle(conf)
			if c.String("database") == "" {
				fmt.Println("Resetting all databases")
				return cleanAnalysisAll(conf, dbm)
			}
			fmt.Println("Resetting database:", c.String("database"))
			return cleanAnalysis(c.String("database"), conf, dbm)
		},
	}

	bootstrapCommands(reset)
}

// cleanAnalysis cleans out all of the analysis data, leaving behind only the
// raw data from parsing the logs
func cleanAnalysis(database string, conf *config.Resources, dbm *database.MetaDBHandle) error {
	//clean database

	conn := conf.System.StructureConfig.ConnTable
	http := conf.System.StructureConfig.HttpTable
	dns := conf.System.DnsConfig.DnsTable

	names, err := conf.Session.DB(database).CollectionNames()
	if err != nil || len(names) == 0 {
		fmt.Fprintf(os.Stderr, "Failed to find analysis results\n")
		return err
	}

	//check if we had an issue dropping a collection
	var err2Flag error = nil
	for _, name := range names {
		switch name {
		case conn, http, dns:
			continue
		default:
			err2 := conf.Session.DB(database).C(name).DropCollection()
			if err2 != nil {
				fmt.Fprintf(os.Stderr, "Failed to drop collection: %s\n", err2.Error())
				err2Flag = err2
			}
		}
	}

	//change metadb
	err3 := dbm.MarkDBCompleted(database, false)

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
func cleanAnalysisAll(conf *config.Resources, dbm *database.MetaDBHandle) error {
	var err error = nil

	for _, name := range dbm.GetAnalyzedDatabases() {
		e := cleanAnalysis(name, conf, dbm)
		//return last error
		if e != nil {
			err = e
		}
	}
	return err
}
