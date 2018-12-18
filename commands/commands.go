package commands

import (
	"runtime"
	"strconv"

	"github.com/urfave/cli"
)

var (
	allCommands []cli.Command

	// below are some prebuilt flags that get used often in various commands

	// threadFlag allows users to specify how many threads should be used
	threadFlag = cli.IntFlag{
		Name:  "threads, t",
		Usage: "Use `N` threads when executing this command",
		Value: runtime.NumCPU(),
	}

	// configFlag allows users to specify an alternate config file to use
	configFlag = cli.StringFlag{
		Name:  "config, c",
		Usage: "Use a given `CONFIG_FILE` when running this command",
		Value: "",
	}

	// for output we often want a human readable option which produces a nice
	// report instead of the simple csv style output
	humanFlag = cli.BoolFlag{
		Name:  "pretty, P",
		Usage: "Print a report instead of csv",
	}

	blSortFlag = cli.StringFlag{
		Name:  "sort, s",
		Usage: "Sort by conn (# of connections), uconn (# of unique connections), total_bytes (# of bytes)",
		Value: "conn",
	}

	blConnFlag = cli.BoolFlag{
		Name:  "connected, C",
		Usage: "Show hosts which were connected to this blacklisted entry",
	}
)

// bootstrapCommands simply adds a given command to the allCommands array
func bootstrapCommands(commands ...cli.Command) {
	for _, command := range commands {
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
