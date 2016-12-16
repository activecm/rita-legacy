package commands

import (
	"fmt"
	"os"

	"github.com/bglebrun/rita/config"
	"github.com/bglebrun/rita/database"
	"github.com/bglebrun/rita/intel"
	"github.com/urfave/cli"
)

func init() {
	analyzeCommand := cli.Command{
		Name:  "analyze",
		Usage: "Analyze imported databases, if no [database,d] flag is specified will attempt all",
		Flags: []cli.Flag{
			databaseFlag,
			verboseFlag,
		},
		Action: func(c *cli.Context) error {
			analyze(c.String("database"), globalVerboseFlag)
			return nil
		},
	}

	bootstrapCommands(analyzeCommand)
}

func analyze(inDb string, verboseFlag bool) {

	conf := config.InitConfig("")
	var torun []string

	// Check to see if we want to run a full database or just one off the command line
	if inDb == "" {
		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"Running analysis against %s \n",
				conf.System.BroConfig.MetaDB)
		}

		ssn := conf.Session.Copy()
		defer ssn.Close()
		iter := ssn.DB(conf.System.BroConfig.MetaDB).
			C("databases").Find(nil).Iter()
		var tdb database.DBMetaInfo

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"Preparing to analyze these databases:\n")
		}

		for iter.Next(&tdb) {
			if tdb.Analysed {
				continue
			}

			if verboseFlag {
				fmt.Fprintf(os.Stdout, "%s \n", tdb.Name)
			}

			torun = append(torun, tdb.Name)
		}
	} else {

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"Preparing to analyze %s\n",
				inDb)
		}
		torun = append(torun, inDb)
	}

	dbm := database.NewMetaDBHandle(conf)

	// TODO: Thread this
	for _, td := range torun {

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"[!] Constructing databases for %s\n", td)
		}
		conf.System.DB = td
		d := database.NewDB(conf)

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Building the Unique Connections Collection\n")
		}

		d.BuildUniqueConnectionsCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Building the Hosts Collection\n")
		}
		d.BuildHostsCollection()

		// The intel module leans on the hostnames collection

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Building the URL collection\n")
		}

		d.BuildUrlsCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Building the hostnames collection\n")
		}

		d.BuildHostnamesCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Adding new external hosts to the intelligence database\n")
		}

		itl := intel.NewIntelHandle(conf)
		itl.Run()

		// Module Collections

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Running beacon analysis\n")
		}

		d.BuildTBDCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Running blacklisted analysis\n")
		}

		d.BuildBlacklistedCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Running useragent analysis\n")
		}

		d.BuildUserAgentCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Searching for port scanning activity\n")
		}

		d.BuildScanningCollection()

		dbm.MarkDBCompleted(td)

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"[+] %s analysis complete!\n", td)
		}

	}
}
