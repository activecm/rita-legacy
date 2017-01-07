package commands

import (
	"fmt"

	"github.com/ocmdev/rita/config"
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
			conf := config.InitConfig("")
			dbm := database.NewMetaDBHandle(conf)
			if c.String("database") == "" {
				fmt.Println("Specify a database with -d")
				return nil
			}
			fmt.Println("Deleting database:", c.String("database"))
			return dbm.DeleteDB(c.String("database"))
		},
	}

	bootstrapCommands(reset)
}
