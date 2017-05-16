package commands

import (
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/parser3"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:  "test-new-parser",
		Usage: "test the new parser",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "index, i",
				Value: 1,
				Usage: "Controls the number of indexing threads",
			},
			cli.IntFlag{
				Name:  "parse, p",
				Value: 1,
				Usage: "Controls the number of parsing threads",
			},
			cli.IntFlag{
				Name:  "buffer, b",
				Value: 50,
				Usage: "Controls the size of the buffer per collection",
			},
		},
		Action: testParser3,
	}

	allCommands = append(allCommands, command)
}

func testParser3(c *cli.Context) error {
	res := database.InitResources(c.String("config"))
	datastore := parser3.NewMongoDatastore(res.DB.Session, c.Int("buffer"), res.Log)
	importer := parser3.NewFSImporter(res, c.Int("index"), c.Int("parse"))
	importer.Run(datastore)
	return nil
}
