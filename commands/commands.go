package commands

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli"
)

var (
	allCommands []cli.Command

	// The destination of the verbose switch
	globalVerboseFlag bool

	// below are some prebuilt flags that get used often in various commands

	verboseFlag = cli.BoolFlag{
		Name:        "verbose, v",
		Usage:       "generate output with timings as command executes",
		Destination: &globalVerboseFlag,
	}

	// databaseFlag allows users to specify which database they'd like to use
	databaseFlag = cli.StringFlag{
		Name:  "dataset, d",
		Usage: "execute this command against `DATASET`",
		Value: "",
	}

	// threadFlag allows users to specify how many threads should be used
	threadFlag = cli.IntFlag{
		Name:  "threads, t",
		Usage: "use `N` threads when executing this command",
		Value: 8,
	}

	// configFlag allows users to specify an alternate config file to use
	configFlag = cli.StringFlag{
		Name:  "config, c",
		Usage: "use `CONFIGFILE` when as configuration when running this command",
		Value: "",
	}
)

// bootstrapCommands simply adds a given command to the allCommands array
func bootstrapCommands(commands ...cli.Command) {
	for _, command := range commands {
		allCommands = append(allCommands, command)
	}
}

// runVerbose runs a given function with with a message and time info
func runVerbose(message string, f func()) {
	startTime := time.Now()
	fmt.Fprintf(os.Stdout, "%s\n", message)
	f()
	fmt.Fprintf(os.Stdout, "completed in %v\n", time.Since(startTime))
}

// Commands provides all of the defined commands to the front end
func Commands() []cli.Command {
	newCommands := []cli.Command{

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
