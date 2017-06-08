package commands

import (
	"time"

	"github.com/ocmdev/rita/analysis/beacon"
	"github.com/ocmdev/rita/analysis/blacklisted"
	"github.com/ocmdev/rita/analysis/crossref"
	"github.com/ocmdev/rita/analysis/dns"
	"github.com/ocmdev/rita/analysis/scanning"
	"github.com/ocmdev/rita/analysis/structure"
	"github.com/ocmdev/rita/analysis/urls"
	"github.com/ocmdev/rita/analysis/useragent"
	"github.com/ocmdev/rita/database"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func init() {
	analyzeCommand := cli.Command{
		Name:  "analyze",
		Usage: "Analyze imported databases, if no [database,d] flag is specified will attempt all",
		Flags: []cli.Flag{
			databaseFlag,
			configFlag,
		},
		Action: func(c *cli.Context) error {
			analyze(c.String("database"), c.String("config"))
			return nil
		},
	}

	bootstrapCommands(analyzeCommand)
}

func analyze(inDb string, configFile string) {
	res := database.InitResources(configFile)
	var toRun []string

	// Check to see if we want to run a full database or just one off the command line
	if inDb == "" {
		res.Log.Info("Running analysis against all databases")
		toRun = append(toRun, res.MetaDB.GetUnAnalyzedDatabases()...)
	} else {
		info, err := res.MetaDB.GetDBMetaInfo(inDb)
		if err != nil {
			res.Log.Errorf("Error: %s not found.\n", inDb)
			return
		}
		if info.Analyzed {
			res.Log.Errorf("Error: %s is already analyzed.\n", inDb)
			return
		}

		toRun = append(toRun, inDb)
	}

	startAll := time.Now()

	res.Log.WithFields(log.Fields{
		"databases":  toRun,
		"start_time": startAll.Format("2006-01-02 15:04:05"),
	}).Info("Preparing to analyze ")

	for _, td := range toRun {
		startIndiv := time.Now()
		res.Log.WithFields(log.Fields{
			"database":   td,
			"start_time": startIndiv.Format("2006-01-02 15:04:05"),
		}).Info("Analyzing")
		res.DB.SelectDB(td)
		logAnalysisFunc("Unique Connections", td, res,
			structure.BuildUniqueConnectionsCollection,
		)
		logAnalysisFunc("Unique Hosts", td, res,
			structure.BuildHostsCollection,
		)
		logAnalysisFunc("Unique Hostnames", td, res,
			dns.BuildHostnamesCollection,
		)
		logAnalysisFunc("Exploded DNS", td, res,
			dns.BuildExplodedDNSCollection,
		)
		logAnalysisFunc("URL Length", td, res,
			urls.BuildUrlsCollection,
		)
		logAnalysisFunc("User Agent", td, res,
			useragent.BuildUserAgentCollection,
		)
		logAnalysisFunc("Beaconing", td, res,
			beacon.BuildBeaconCollection,
		)
		logAnalysisFunc("Blacklisted", td, res,
			blacklisted.BuildBlacklistedCollection,
		)
		logAnalysisFunc("Scanning", td, res,
			scanning.BuildScanningCollection,
		)
		logAnalysisFunc("Cross Reference", td, res,
			crossref.BuildXRefCollection,
		)

		res.MetaDB.MarkDBAnalyzed(td, true)
		endIndiv := time.Now()
		res.Log.WithFields(log.Fields{
			"database": td,
			"end_time": endIndiv.Format("2006-01-02 15:04:05"),
			"duration": endIndiv.Sub(startIndiv),
		}).Info("Analysis complete")
	}
	endAll := time.Now()
	res.Log.WithFields(log.Fields{
		"end_time": endAll.Format("2006-01-02 15:04:05"),
		"duration": endAll.Sub(startAll),
	}).Info("Analysis complete")
}

func logAnalysisFunc(analysisName string, databaseName string,
	resources *database.Resources, analysis func(*database.Resources)) {
	analysisName += " Analysis"
	start := time.Now()
	resources.Log.WithFields(log.Fields{
		"analysis":   analysisName,
		"database":   databaseName,
		"start_time": start.Format("2006-01-02 15:04:05"),
	}).Infof("Running analysis")
	analysis(resources)
	end := time.Now()
	resources.Log.WithFields(log.Fields{
		"analysis": analysisName,
		"database": databaseName,
		"end_time": end.Format("2006-01-02 15:04:05"),
		"duration": end.Sub(start),
	}).Infof("Analysis complete")
}
