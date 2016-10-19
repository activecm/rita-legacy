package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/mgo.v2/bson"

	"github.com/ocmdev/rita/config"
	"github.com/urfave/cli"
)

var beaconHeader = false

func init() {
	command := cli.Command{
		Name:  "show-beacons",
		Usage: "print beacon information to standard out",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:        "no-header, r",
				Usage:       "turn off the header row",
				Destination: &beaconHeader,
			},
			humanFlag,
			databaseFlag,
		},
		Action: showBeacons,
	}

	bootstrapCommands(command)
}

type bcnresult struct {
	Src           string        `bson:"src"`
	Dst           string        `bson:"dst"`
	UconnID       bson.ObjectId `bson:"uconn_id"`
	Range         int64         `bson:"range"`
	Size          int64         `bson:"size"`
	RangeVals     string        `bson:"range_vals"`
	Fill          float64       `bson:"fill"`
	Spread        float64       `bson:"spread"`
	Sum           int64         `bson:"range_size"`
	Score         float64       `bson:"score"`
	TopInterval   int64         `bson:"most_frequent_interval"`
	TopIntervalCt int64         `bson:"most_frequent_interval_count"`
	RangeMin      string
	RangeMax      string
}

// The following is so that we can sort each of the beacons on thier score
type bcnresultset []bcnresult

func (slice bcnresultset) Len() int {
	return len(slice)
}

func (slice bcnresultset) Less(i, j int) bool {
	return slice[i].Score < slice[j].Score
}

func (slice bcnresultset) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// showBeacons shows all beacons for a given database
func showBeacons(c *cli.Context) error {

	if c.String("dataset") == "" {
		return cli.NewExitError("No dataset was not specified", -1)
	}

	if humanreadable {
		return showBeaconsReport(c)
	}

	return showBeaconCsv(c)

}

func showBeaconCsv(c *cli.Context) error {
	tmpl := "{{.Score}},{{.Src}},{{.Dst}},{{.Range}},{{.Size}},{{.RangeMin}},{{.RangeMax}},"
	tmpl += "{{.Fill}},{{.Spread}},{{.Sum}},{{.TopInterval}},{{.TopIntervalCt}}\n"

	out, err := template.New("tbd").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")
	if c.String("dataset") != "" {
		conf.System.DB = c.String("dataset")
	}

	coll := conf.Session.DB(conf.System.DB).C(conf.System.TBDConfig.TBDTable)
	iter := coll.Find(nil).Iter()

	var res bcnresult
	var allres bcnresultset

	for iter.Next(&res) {
		ranges := strings.Split(res.RangeVals, "--")
		if len(ranges) == 1 {
			res.RangeMin = ranges[0]
			res.RangeMax = ranges[0]
		} else {
			res.RangeMin = ranges[0]
			res.RangeMax = ranges[1]
		}
		allres = append(allres, res)
	}

	sort.Sort(allres)

	for _, cres := range allres {
		err := out.Execute(os.Stdout, cres)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}

	return nil

}

func showBeaconsReport(c *cli.Context) error {

	if !c.Bool("no-header") {
		hdr := "score\t          source\t            dest\trange\tsize\trange-min\trange-max\t  fill\tspread\t   sum\ttop-interval\ttop-interval-cnt\n"
		hdr += "-----\t----------------\t----------------\t-----\t----\t---------\t---------\t------\t------\t------\t------------\t----------------\n"
		fmt.Fprintf(os.Stdout, hdr)
	}

	tmpl := `{{.Score | printf "%.3f"}}`
	tmpl += "\t{{.Src}}\t{{.Dst}}\t"
	tmpl += `{{.Range | printf "%5d"}}`
	tmpl += "\t" + `{{.Size | printf "%5d"}}` + "\t"
	tmpl += "{{.RangeMin}}\t{{.RangeMax}}\t" + `{{.Fill | printf "%3.3f"}}`
	tmpl += "\t" + `{{.Spread | printf "%3.3f"}}` + "\t" + `{{.Sum | printf "%6d"}}`
	tmpl += "\t" + `{{.TopInterval | printf "%12d"}}` + "\t" + `{{.TopIntervalCt | printf "%16d"}}`
	tmpl += "\n"

	out, err := template.New("tbd").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")
	if c.String("dataset") != "" {
		conf.System.DB = c.String("dataset")
	}

	coll := conf.Session.DB(conf.System.DB).C(conf.System.TBDConfig.TBDTable)
	iter := coll.Find(nil).Iter()

	var res bcnresult
	var allres bcnresultset

	for iter.Next(&res) {
		res = processBeaconbcnresult(res)
		allres = append(allres, res)
	}

	sort.Sort(allres)

	for _, cres := range allres {
		err := out.Execute(os.Stdout, cres)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}

	return nil
}

func processBeaconbcnresult(r bcnresult) bcnresult {
	r.Src = padAddr(r.Src)
	r.Dst = padAddr(r.Dst)
	mm := parseRangeVals(r.RangeVals)
	r.RangeMin = mm[0]
	r.RangeMax = mm[1]
	return r
}

func parseRangeVals(rvals string) []string {
	minmax := strings.Split(rvals, "--")
	minmax[0] = leftPad(minmax[0], 8)
	if len(minmax) == 1 {
		minmax = append(minmax, minmax[0])
		return minmax
	}
	minmax[1] = leftPad(minmax[1], 8)
	return minmax
}
