package commands

import (
	"fmt"

	"github.com/bglebrun/rita/config"
	"github.com/bglebrun/rita/database"
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
			conf := config.InitConfig("")
			dbm := database.NewMetaDBHandle(conf)
			if c.String("database") == "" {
				return cli.NewExitError("Specify a database with -d", -1)
			}
			fmt.Println("Deleting database:", c.String("database"))
			return dbm.DeleteDB(c.String("database"))
		},
	}

	bootstrapCommands(reset)
}
