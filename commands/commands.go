package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli"
)

var (
	allCommands []cli.Command

	// The destination of the verbose switch
	globalVerboseFlag bool

	// For the human readable flags
	humanreadable = false

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

	// for output we often want a human readable option which produces a nice
	// report instead of the simple csv style output
	humanFlag = cli.BoolFlag{
		Name:        "human-readable, H",
		Usage:       "print a report instead of csv",
		Destination: &humanreadable,
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

// padAddr specifically pads ip addresses so that they're a full 16 chars wide
func padAddr(addr string) string {
	for {

		if len(addr) > 15 {
			return addr
		}
		addr = " " + addr
	}
}

// leftPad
func leftPad(s string, n int) string {
	for {
		s = " " + s
		if len(s) > n {
			return s
		}
	}
}
