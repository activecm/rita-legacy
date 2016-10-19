package commands

import (
	"fmt"
	"os"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/ocmdev/rita/config"
	"github.com/urfave/cli"
)

func init() {
	resetdb := cli.Command{
		Name:  "reset-database",
		Usage: "reset analysis of a particular database",
		Flags: []cli.Flag{
			databaseFlag,
		},
		Action: func(c *cli.Context) error {
			if c.String("dataset") == "" {
				fmt.Fprintf(os.Stderr, "please specify a database\n")
				os.Exit(-1)
			}
			fmt.Println("Warning: this will not reset the analyzed flag in metadb")

			cleanAnalysis(c.String("database"))
			return nil
		},
	}

	resettest := cli.Command{
		Name:  "reset-test",
		Usage: "reset analysis of a particular test",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "test, t",
				Usage: "Remove analysis collections for `TEST`",
				Value: "",
			},
		},
		Action: func(c *cli.Context) error {
			if c.String("test") == "" {
				fmt.Fprintf(os.Stderr, "please specify a test\n")
				os.Exit(-1)
			}
			fmt.Println("Resetting test:", c.String("test"))
			cleanAnalysisAll(c.String("test"))
			return nil
		},
	}

	bootstrapCommands(resetdb, resettest)
}

// cleanAnalysis cleans out all of the analysis data, leaving behind only the
// raw data from parsing the logs
func cleanAnalysis(dataset string) {
	conf := config.InitConfig("")
	conf.System.DB = dataset

	conn := conf.System.StructureConfig.ConnTable
	http := conf.System.StructureConfig.HttpTable
	dns := conf.System.DnsConfig.DnsTable
	names, err := conf.Session.DB(dataset).CollectionNames()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get collection names: %s\n", err.Error())
		os.Exit(-1)
	}

	for _, name := range names {
		switch name {
		case conn, http, dns:
			continue
		default:
			err := conf.Session.DB(dataset).C(name).DropCollection()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to drop collection: %s\n", err.Error())
			}
		}
	}

}

// cleanAnalysisAll uses the metadb to walk all databases and clean the analysis
func cleanAnalysisAll(dataset string) {
	conf := config.InitConfig("")
	conf.System.DB = dataset

	coll := conf.Session.DB(dataset).C("databases")
	iter := coll.Find(nil).Iter()

	var dbinfo struct {
		ID       bson.ObjectId `bson:"_id"`
		Name     string        `bson:"name"`
		Analyzed bool          `bson:"analyzed"`
	}

	for iter.Next(&dbinfo) {
		if dbinfo.Analyzed {
			cleanAnalysis(dbinfo.Name)
			change := mgo.Change{
				Update:    bson.M{"$set": bson.M{"analyzed": false}},
				ReturnNew: true,
			}
			_, err := coll.Find(bson.M{"_id": dbinfo.ID}).Apply(change, &dbinfo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to update metadb: %s\n", err.Error())
			}
			if dbinfo.Analyzed {
				fmt.Fprintf(os.Stderr, "Warning %s may not have updated in meta.\n", dbinfo.Name)
			}
		}
	}
}
