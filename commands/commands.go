package commands

import (
	"runtime"
	"strconv"

	"github.com/activecm/rita/resources"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	allCommands []cli.Command

	// below are some prebuilt flags that get used often in various commands

	// configFlag allows users to specify an alternate config file to use
	configFlag = cli.StringFlag{
		Name:  "config, c",
		Usage: "Use a given `CONFIG_FILE` when running this command",
		Value: "",
	}

	// forceFlag allows users to bypass prompts
	forceFlag = cli.BoolFlag{
		Name:  "force, f",
		Usage: "Bypass verification prompt",
	}

	// resetFlag allows automatic reset of analysis prior to the command
	resetFlag = cli.BoolFlag{
		Name:  "reset, r",
		Usage: "Reset database analysis",
	}

	rollingFlag = cli.BoolFlag{
		Name:  "rolling, R",
		Usage: "Indicates rolling import, which builds on and removes data to maintain a rolling 24-hour analysis",
	}

	// for rolling analysis: says how many chunks are in a given day
	totalChunksFlag = cli.IntFlag{
		Name:  "numchunks, NC",
		Usage: "For rolling analysis: How many chunks are in a given day, with import frequency being every (24/numchunks) hours. Number must be a multiple of 24. (Example: 12 = import every 2 hrs, 24 = every hour, 6 = every 4 hrs)",
		Value: -1,
	}

	// for rolling analysis: says this is the n-th chunk of the day (the first
	//  being midnight-1:59:59AM, the second being 2am-3:59:59am, etc, depending
	// on 24/number of total chunks (in the flag above)
	currentChunkFlag = cli.IntFlag{
		Name:  "chunk, CC",
		Usage: "For rolling analysis: This is the `N`th chunk of the day",
		Value: -1,
	}

	// threadFlag allows users to specify how many threads should be used
	threadFlag = cli.IntFlag{
		Name:  "threads, t",
		Usage: "Use `N` threads when executing this command",
		Value: runtime.NumCPU(),
	}

	// for output we often want a human readable option which produces a nice
	// report instead of the simple csv style output
	humanFlag = cli.BoolFlag{
		Name:  "human-readable, H",
		Usage: "Print a report instead of csv",
	}

	blSortFlag = cli.StringFlag{
		Name:  "sort, s",
		Usage: "Sort by conn_count (# of connections), uconn_count (# of unique connections), total_bytes (# of bytes)",
		Value: "conn_count",
	}

	blConnFlag = cli.BoolFlag{
		Name:  "connected, C",
		Usage: "Show hosts which were connected to this blacklisted entry",
	}
)

// bootstrapCommands simply adds a given command to the allCommands array
func bootstrapCommands(commands ...cli.Command) {
	for _, command := range commands {
		command.Before = func(c *cli.Context) error {
			//Get access to the logger
			configFile := c.String("config")
			res := resources.InitResources(configFile)
			//Display args in logs
			fields := log.Fields{
				"Arguments": c.Args(),
			}
			//Display flag info in logs
			for _, it := range c.GlobalFlagNames() {
				if c.IsSet(it) {
					fields["Global Flag("+it+")"] = c.GlobalGeneric(it)
				}
			}
			for _, it := range c.FlagNames() {
				if c.IsSet(it) {
					fields["Flag("+it+")"] = c.Generic(it)
				}
			}
			res.Log.WithFields(fields).Info("Running Command: " + command.Name)
			return nil
		}
		allCommands = append(allCommands, command)
	}
}

// Commands provides all of the defined commands to the front end
func Commands() []cli.Command {
	return allCommands
}

//helper functions for formatting floats and integers
func f(f float64) string {
	return strconv.FormatFloat(f, 'g', 6, 64)
}
func i(i int64) string {
	return strconv.FormatInt(i, 10)
}
