package commands

import (
	"fmt"
	"time"

	"github.com/activecm/rita/analysis/blacklist"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/blang/semver"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func init() {
	analyzeCommand := cli.Command{
		Name:  "analyze",
		Usage: "Analyze imported databases",
		UsageText: "rita analyze [command options] [database]\n\n" +
			"If no database is specified, every database will be analyzed.",
		Flags: []cli.Flag{
			resetFlag,
			configFlag,
		},
		Action: func(c *cli.Context) error {
			// set database - if empty string, all databases will be analyzed
			database := c.Args().Get(0)

			// set config file - if empty string default will be used
			configFile := c.String("config")

			// set resources from config file
			res := resources.InitResources(configFile)

			// set reset flags
			reset := c.Bool("reset")

			// check current version
			fmt.Printf(updateCheck(configFile))

			// if the reset flag was given reset analysis before analyzing
			if reset {
				fmt.Println("[+] Resetting:")
				// Check which database(s) need to be reset
				if database == "" {
					for _, entry := range res.MetaDB.GetDatabases() {
						_ = resetAnalysis(entry, res, true)
					}
				} else {
					_ = resetAnalysis(database, res, true)
				}
			}

			// run analysis
			err := analyze(database, res, reset)

			return err
		},
	}

	bootstrapCommands(analyzeCommand)
}

func analyze(inDb string, res *resources.Resources, resetFlag bool) error {

	var toRunDirty []string
	var toRun []string

	// Check to see if we want to run a full database or just one off the command line
	if inDb == "" {
		res.Log.Info("Running analysis against all databases")
		toRunDirty = append(toRun, res.MetaDB.GetAnalyzeReadyDatabases()...)
	} else {
		toRunDirty = append(toRun, inDb)
	}

	// Check for problems
	for _, possDB := range toRunDirty {
		info, err := res.MetaDB.GetDBMetaInfo(possDB)
		if err != nil {
			errStr := fmt.Sprintf("Error: %s not found.", possDB)
			res.Log.Errorf(errStr)
			fmt.Println(errStr)
			continue
		}
		if !info.ImportFinished {
			errStr := fmt.Sprintf("Error: %s hasn't finished being imported.", possDB)
			res.Log.Errorf(errStr)
			fmt.Println(errStr)
			continue
		}
		if info.Analyzed {
			errStr := fmt.Sprintf("Error: %s is already analyzed.", possDB)
			res.Log.Errorf(errStr)
			fmt.Println(errStr)
			continue
		}
		semVer, err := semver.ParseTolerant(info.ImportVersion)
		if err != nil {
			errStr := fmt.Sprintf("Error: %s is labelled with an incorrect version tag", possDB)
			res.Log.Errorf(errStr)
			fmt.Println(errStr)
			continue
		}
		if semVer.Major != res.Config.R.Version.Major {
			errStr := fmt.Sprintf("Error: %s was parsed by an incompatible version of RITA", possDB)
			res.Log.Errorf(errStr)
			fmt.Println(errStr)
			continue
		}
		toRun = append(toRun, possDB)
	}

	startAll := time.Now()

	if len(toRun) > 0 {
		fmt.Println("[+] Analyzing:")
	}

	for _, db := range toRun {
		fmt.Println("\t[-] " + db)
	}
	res.Log.WithFields(log.Fields{
		"databases":  toRun,
		"start_time": startAll.Format(util.TimeFormat),
	}).Info("Preparing to analyze ")

	for _, td := range toRun {
		startIndiv := time.Now()
		res.Log.WithFields(log.Fields{
			"database":   td,
			"start_time": startIndiv.Format(util.TimeFormat),
		}).Info("Analyzing")
		fmt.Println("[+] Analyzing " + td)
		res.DB.SelectDB(td)

		if res.Config.S.Blacklisted.Enabled {
			logAnalysisFunc("Blacklisted", td, res,
				blacklist.BuildBlacklistedCollections,
			)
		}

		res.MetaDB.MarkDBAnalyzed(td, true)
		endIndiv := time.Now()
		res.Log.WithFields(log.Fields{
			"database": td,
			"end_time": endIndiv.Format(util.TimeFormat),
			"duration": endIndiv.Sub(startIndiv),
		}).Info("Analysis complete")
	}
	endAll := time.Now()
	res.Log.WithFields(log.Fields{
		"end_time": endAll.Format(util.TimeFormat),
		"duration": endAll.Sub(startAll),
	}).Info("Analysis complete")
	return nil
}

func logAnalysisFunc(analysisName string, databaseName string,
	resources *resources.Resources, analysis func(*resources.Resources)) {
	analysisName += " Analysis"
	start := time.Now()
	resources.Log.WithFields(log.Fields{
		"analysis":   analysisName,
		"database":   databaseName,
		"start_time": start.Format(util.TimeFormat),
	}).Infof("Running analysis")
	fmt.Println("\t[-] Running " + analysisName)
	analysis(resources)
	end := time.Now()
	resources.Log.WithFields(log.Fields{
		"analysis": analysisName,
		"database": databaseName,
		"end_time": end.Format(util.TimeFormat),
		"duration": end.Sub(start),
	}).Infof("Analysis complete")
}
