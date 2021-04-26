package commands

import (
	"fmt"

	"github.com/activecm/rita/resources"
	"github.com/urfave/cli"
)

func init() {

	databases := cli.Command{
		Name:    "list",
		Aliases: []string{"show-databases"},
		Usage:   "Print the databases currently stored",
		Flags: []cli.Flag{
			ConfigFlag,
		},
		Action: func(c *cli.Context) error {
			res := resources.InitResources(getConfigFilePath(c))

			if res != nil {
				for _, name := range res.MetaDB.GetDatabases() {
					fmt.Println(name)
				}
			} else {
				fmt.Println("\t[-] Cannot display databases due to outdated metadatabase entries.")
			}

			return nil
		},
	}

	bootstrapCommands(databases)
}
