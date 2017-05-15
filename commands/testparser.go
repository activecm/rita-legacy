package commands

import (
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/parser3"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:   "test-new-parser",
		Usage:  "test the new parser",
		Action: testParser3,
	}

	allCommands = append(allCommands, command)
}

func testParser3(c *cli.Context) error {
	res := database.InitResources(c.String("config"))
	datastore := parser3.NewMongoDatastore()
	importer := parser3.NewFSImporter(res, 1, 1)
	importer.Run(datastore)
	return nil
}
