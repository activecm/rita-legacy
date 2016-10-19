#RITA Commands

###Hacking

####Some things to remember
Commands should behave like typical Unix systems commands. For example, the human readable output from
show-beacons or show-scans is MUCH easier to read... but the default is just an unformatted comma
delimited list which looks like garbage. The latter option is much easier to parse with sed, awk, cut, etc...

We've tried to make adding commands to rita easier. Here's a quick rundown of how a command might be added
to the system.

1. Create a new command file in the commands directory called "nameofcommand.go"
1. Create an init function in this command that declares your command and adds it to the allCommands global.
1. Create a function that executes the business logic of your command.
1. Profit.
```go
func init() {
	// command to do something
	command := cli.Command{
		Flags: []cli.Flags{
			cli.IntFlag{
				Name: "test, t",
				Usage: "set test flag",
				Value: 29,
			},
			// There are also a few pre-defined flags for you to use
			configFlag,
		},
		Name: "nameofcommand",
		Usage: "how to use the command",
		Action: nameOfCmdFunc,
	}

	// command to do some related thing
	othercommand := cli.Command{
		Flags: []cli.Flags{
			cli.IntFlag{
				Name: "test, t",
				Usage: "set test flag",
				Value: 29,
			},
			// There are also a few pre-defined flags for you to use
			configFlag,
		},
		Name: "nameofothercommand",
		Usage: "how to use the other command",
		Action: nameOfOtherCmdFunc,
	}

	// Add the command to the allCommands data structure (IMPORTANT)
	bootstrapCommands(command, othercommand)
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
