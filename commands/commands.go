package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/parser"
	"github.com/ocmdev/rita/parser/docwriter"

	"github.com/urfave/cli"
)

// verboseFlag is present for commands that support verbose mode
var verboseFlag bool

var allCommands []cli.Command

// Commands provides all of the defined commands to the front end
func Commands() []cli.Command {
	newCommands := []cli.Command{
		{
			Name:  "import",
			Usage: "uses the configured bro importer to import files",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "threads, t",
					Usage: "number of write threads to use",
					Value: 18,
				},
				cli.StringFlag{
					Name:  "config, c",
					Usage: "specify a config file to be used",
					Value: "",
				},
			},

			Action: func(c *cli.Context) error {
				conf := config.InitConfig(c.String("config"))
				metadb := database.NewMetaDBHandle(conf)

				dw := docwriter.New(conf, metadb)
				dw.Start(c.Int("threads"))
				parser.NewWatcher(conf, metadb).Run(dw)

				return nil
			},
		},
		{
			Name:  "analyze",
			Usage: "Analyze imported databases, if no [database,d] flag is specified will attempt all",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "database, d",
					Usage: "run against `DATABASE`",
					Value: "",
				},
				cli.BoolFlag{
					Name:        "verbose, v",
					Usage:       "print status to stdout",
					Destination: &verboseFlag,
				},
			},
			Action: func(c *cli.Context) error {
				analyze(c.String("database"), verboseFlag)
				return nil
			},
		},
		{
			Name:  "show-beacons",
			Usage: "print beacon information to standard out",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "database, d",
					Usage: "print beacons for `DATABASE`",
					Value: "",
				},
			},
			Action: func(c *cli.Context) error {
				if c.String("database") == "" {
					return errors.New("No database specified")
				}
				showBeacons(c.String("database"))
				return nil
			},
		},
		{
			Name:  "show-scans",
			Usage: "print scanning information to standard out",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "database, d",
					Usage: "print scans for `DATABASE`",
					Value: "",
				},
			},
			Action: func(c *cli.Context) error {
				if c.String("database") == "" {
					return errors.New("No database specified")
				}
				showScans(c.String("database"))
				return nil
			},
		},
		{
			Name:  "show-blacklisted",
			Usage: "print blacklisted information to standard out",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "database, d",
					Usage: "print scans for `DATABASE`",
					Value: "",
				},
			},
			Action: func(c *cli.Context) error {
				if c.String("database") == "" {
					return errors.New("No database specified")
				}
				showBlacklisted(c.String("database"))
				return nil
			},
		},

		{
			Name:  "reset-database",
			Usage: "reset analysis of a particular database",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "database, d",
					Usage: "Remove analysis collections from `DATABASE`",
					Value: "",
				},
			},
			Action: func(c *cli.Context) error {
				if c.String("database") == "" {
					fmt.Fprintf(os.Stderr, "please specify a database\n")
					os.Exit(-1)
				}
				fmt.Println("Warning: this will not reset the analyzed flag in metadb")

				cleanAnalysis(c.String("database"))
				return nil
			},
		},

		{
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
		},
	}

	for _, command := range allCommands {
		newCommands = append(newCommands, command)
	}
	return newCommands
}
