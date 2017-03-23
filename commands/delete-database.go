package commands

import (
	"fmt"

	"github.com/ocmdev/rita/database"
	"github.com/urfave/cli"
)

func init() {
	reset := cli.Command{
		Name:  "delete-database",
		Usage: "Delete an imported database",
		Flags: []cli.Flag{
			databaseFlag,
		},
		Action: func(c *cli.Context) error {
			res := database.InitResources("")
			if c.String("database") == "" {
				return cli.NewExitError("Specify a database with -d", -1)
			}
			fmt.Println("Deleting database:", c.String("database"))
			return res.MetaDB.DeleteDB(c.String("database"))
		},
	}

	bootstrapCommands(reset)
}
