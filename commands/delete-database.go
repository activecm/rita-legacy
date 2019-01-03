package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/urfave/cli"
)

func init() {
	reset := cli.Command{
		Name:      "delete",
		Aliases:   []string{"delete-database"},
		Usage:     "Delete an imported database",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			forceFlag,
			configFlag,
		},
		Action: func(c *cli.Context) error {
			res := resources.InitResources(c.String("config"))
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}

			return deleteDatabase(db, res, c.Bool("force"))

		},
	}

	bootstrapCommands(reset)
}

//deleteDatabase deletes a target database
func deleteDatabase(database string, res *resources.Resources, forceFlag bool) error {

	// get all database names
	names, err := res.DB.Session.DatabaseNames()

	// check if database exists
	dbExists := util.StringInSlice(database, names)

	// check if metadatabase record for database exists
	mDBExists := util.StringInSlice(database, res.MetaDB.GetDatabases())

	// if no force flag, verify action with the user
	if !forceFlag {
		fmt.Print("Are you sure you want to delete database ", database, " [y/N] ")

		read := bufio.NewReader(os.Stdin)

		response, _ := read.ReadString('\n')

		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			fmt.Println("Deleting database: ", database)
		} else {
			return cli.NewExitError("Database "+database+" was not deleted.", 0)
		}
	}

	// delete database if it exists
	if dbExists {
		if res.DB.Session.DB(database).DropDatabase() != nil {
			return cli.NewExitError("Failed to delete database", -1)
		}
	}

	// remove metadatabase record if it exists
	if mDBExists {
		if res.MetaDB.DeleteDB(database) != nil {
			return cli.NewExitError("Failed to update metadb", -1)
		}
	}

	// check if we have databases
	if err != nil || len(names) == 0 {
		return cli.NewExitError("Failed to find any databases", -1)
	}

	// check if we have anything to delete
	if !dbExists && !mDBExists {
		return cli.NewExitError("No records for database found", -1)
	}

	// if it got here, deleting was a success!
	fmt.Fprintf(os.Stdout, "\t[-] Successfully deleted database %s.\n", database)

	return nil
}
