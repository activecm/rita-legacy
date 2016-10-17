package commands

import (
	"fmt"
	"os"
	"text/template"

	"github.com/ocmdev/rita/datatypes/scanning"

	"github.com/ocmdev/rita/config"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// cleanAnalysis cleans out all of the analysis data, leaving behind only the
// raw data from parsing the logs
func cleanAnalysis(dataset string) {
	conf := config.InitConfig("")
	conf.System.DB = dataset

	conn := conf.System.StructureConfig.ConnTable
	http := conf.System.StructureConfig.HttpTable
	dns := conf.System.DnsConfig.DnsTable
	names, err := conf.Session.DB(dataset).CollectionNames()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get collection names: %s\n", err.Error())
		os.Exit(-1)
	}

	for _, name := range names {
		switch name {
		case conn, http, dns:
			continue
		default:
			err := conf.Session.DB(dataset).C(name).DropCollection()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to drop collection: %s\n", err.Error())
			}
		}
	}

}

// cleanAnalysisAll uses the metadb to walk all databases and clean the analysis
func cleanAnalysisAll(dataset string) {
	conf := config.InitConfig("")
	conf.System.DB = dataset

	coll := conf.Session.DB(dataset).C("databases")
	iter := coll.Find(nil).Iter()

	var dbinfo struct {
		ID       bson.ObjectId `bson:"_id"`
		Name     string        `bson:"name"`
		Analyzed bool          `bson:"analyzed"`
	}

	for iter.Next(&dbinfo) {
		if dbinfo.Analyzed {
			cleanAnalysis(dbinfo.Name)
			change := mgo.Change{
				Update:    bson.M{"$set": bson.M{"analyzed": false}},
				ReturnNew: true,
			}
			_, err := coll.Find(bson.M{"_id": dbinfo.ID}).Apply(change, &dbinfo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to update metadb: %s\n", err.Error())
			}
			if dbinfo.Analyzed {
				fmt.Fprintf(os.Stderr, "Warning %s may not have updated in meta.\n", dbinfo.Name)
			}
		}
	}
}

// showScans prints all scans for a given database
func showScans(dataset string) {

	cols := "source\tdest\tport-count\tports\n"
	tmpl := "{{.Src}}\t{{.Dst}}\t{{.PortSetCount}}\t"
	tmpl += "{{range $idx, $port := .PortSet}} {{ $port }} {{end}}\n"
	out, err := template.New("scn").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")
	conf.System.DB = dataset

	var res scanning.Scan

	coll := conf.Session.DB(dataset).C(conf.System.ScanningConfig.ScanTable)
	iter := coll.Find(nil).Iter()

	fmt.Printf(cols)
	for iter.Next(&res) {
		err := out.Execute(os.Stdout, res)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}

}

// showBlacklisted prints all blacklisted for a given database
func showBlacklisted(dataset string) {

	cols := "host\tscore\n"
	tmpl := "{{.Host}}\t{{.Score}}\n"
	out, err := template.New("bl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")
	conf.System.DB = dataset

	var res scanning.Scan

	coll := conf.Session.DB(dataset).C(conf.System.BlacklistedConfig.BlacklistTable)
	iter := coll.Find(nil).Iter()

	fmt.Printf(cols)
	for iter.Next(&res) {
		err := out.Execute(os.Stdout, res)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}

}
