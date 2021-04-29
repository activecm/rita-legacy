# RITA Commands

## A note on philosophy

As RITA is written in Go and targeted towards Unix based systems, we believe that RITA too should follow the Unix Philosophy.

As such RITA commands should implement this philosophy by default.

For example, the human readable output from `show-beacons` or `show-blacklisted` is easy to read, but the default action is to print an unformatted comma separated list. This "ugly" format is much easier to parse and process with tools such as `sed`, `awk`, and `cut`.

These tools are great at processing the results that come out of RITA, and we believe that RITA should do its best to support them.

## How to create a new command

Adding commands to RITA is a straight-forward process.

1. Create a new command file in the commands directory
1. Create an `init` function in the newly created file that declares your command and boostraps it
1. Create a function that executes the business logic of your command
1. Profit.

## Example Command

```go
//init functions run import
func init() {
	// command to do something
	command := cli.Command{
		Flags: []cli.Flags{
			cli.IntFlag{
				Name: "test, t",
				Usage: "set test flag",
				Value: 29,
			},
			// There are also a few pre-defined flags for you to use in commands.go
			allFlag,
		},
		Name: "name-of-command",
		Usage: "how to use the command",
		Action: nameOfCmdFunc,
	}

	// command to do something else
	otherCommand := cli.Command{
		Flags: []cli.Flags{
			cli.IntFlag{
				Name: "test, t",
				Usage: "set test flag",
				Value: 29,
			},
			ConfigFlag,
		},
		Name: "name-of-other-command",
		Usage: "how to use the other command",
		Action: nameOfOtherCmdFunc,
	}

	// Bootstrap the commands (IMPORTANT)
	bootstrapCommands(command, otherCommand)
}

// It is very important that we use a function of this type (for compatibility with cli)
func nameOfCmdFunc(c *cli.Context) error {
	// do stuff
	return nil
}

// another command function
func nameOfOtherCmdFunc(c *cli.Context) error {
	// do stuff
	return nil
}
```
