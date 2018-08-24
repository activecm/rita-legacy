package commands

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/activecm/rita/resources"
	"github.com/urfave/cli"
)

func init() {
	reset := cli.Command{
		Name:      "delete-database",
		Usage:     "Delete an imported database",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			configFlag,
		},
		Action: func(c *cli.Context) error {
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}

			fmt.Print("Are you sure you want to delete database ", db, " [y/N] ")

			read := bufio.NewReader(os.Stdin)

			response, err := read.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			response = strings.ToLower(strings.TrimSpace(response))

			if response == "y" || response == "yes" {
				res := resources.InitResources(c.String("config"))
				return deleteDatabase(res, db)
			}
			return cli.NewExitError("Database "+db+" was not deleted.", 0)
		},
	}

	bootstrapCommands(reset)
}

func deleteDatabase(res *resources.Resources, db string) error {
	fmt.Println("Deleting database:", db)
	ritaDB, err := res.DBIndex.GetDatabase(db)
	if err != nil {
		return cli.NewExitError("Error: could not delete database: "+err.Error(), -1)
	}
	err = res.FileIndex.RemoveFilesForDatabase(db)
	if err != nil {
		return cli.NewExitError("Error: could not delete database: "+err.Error(), -1)
	}
	err = ritaDB.DeleteIndex(res.DB.Session)
	if err != nil {
		return cli.NewExitError("Error: could not delete database: "+err.Error(), -1)
	}
	err = ritaDB.Drop(res.DB.Session)
	if err != nil {
		return cli.NewExitError("Error: could not delete database: "+err.Error(), -1)
	}
	return nil
}
