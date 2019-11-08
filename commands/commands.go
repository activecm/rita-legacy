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

	// allFlag indicates all databases in option
	allFlag = cli.BoolFlag{
		Name:  "all, a",
		Usage: "Indicates all databases should be removed",
	}

	// matchFlag indicates only matched databases to string SEARCH_STRING
	matchFlag = cli.BoolFlag{
		Name:  "match, m",
		Usage: "Indicate only databases matching <database> string should be removed",
	}

	// regexFlag indicates use of matching regex pattern REGEX_PATTERN
	regexFlag = cli.BoolFlag{
		Name:  "regex, r",
		Usage: "Indicate use regular expression as <database> string to be removed",
	}

	// dryRun indicates which databases would be deleted with current options
	dryRunFlag = cli.BoolFlag{
		Name:  "dry-run, n",
		Usage: "Tests which databases would be deleted. Does not actually delete any data, nor prompt for confirmation",
	}

	// deleteFlag indicates whether any matching, existing data should be deleted
	// before importing the target data
	deleteFlag = cli.BoolFlag{
		Name:  "delete, D",
		Usage: "Indicates that the existing dataset should be deleted before re-importing. If the dataset is a rolling dataset and --chunk is not specified, the latest chunk will be replaced.",
	}

	rollingFlag = cli.BoolFlag{
		Name:  "rolling, R",
		Usage: "Indicates rolling import, which builds on and removes data to maintain a fixed length of time",
	}

	// for rolling analysis: says how many chunks are in a given day
	totalChunksFlag = cli.IntFlag{
		Name:  "numchunks, NC",
		Usage: "Implies --rolling: How many chunks are in a given dataset. This, along with the duration of each chunk will determine the duration of your dataset. E.g. 1hr chunks * 24 chunks is 1 day of data",
		Value: -1,
	}

	currentChunkFlag = cli.IntFlag{
		Name:  "chunk, CC",
		Usage: "Implies --rolling: This is the `N`th chunk of the dataset. `chunk` must be 0 <= `chunk` < `numchunks`",
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

	limitFlag = cli.IntFlag{
		Name:  "limit, li",
		Usage: "Limit the outputs of the result to `LIMIT` values",
		Value: 1000,
	}

	noLimitFlag = cli.BoolFlag{
		Name:  "no-limit, nl",
		Usage: "No limit to the outputs of results",
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
