package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:      "show-long-connections",
		Usage:     "Print long connections and relevant information",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
			configFlag,
			limitFlag,
			noLimitFlag,
			delimFlag,
			netNamesFlag,
		},
		Action: func(c *cli.Context) error {
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}

			res := resources.InitResources(c.String("config"))
			res.DB.SelectDB(db)

			sortStr := "maxdur"
			sortDirection := -1
			thresh := 60 // 1 minute

			data, err := getLongConnsResultsView(res, thresh, sortStr, sortDirection, c.Int("limit"), c.Bool("no-limit"))

			if err != nil {
				res.Log.Error(err)
				return cli.NewExitError(err, -1)
			}

			if !(len(data) > 0) {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showConnsHuman(data, c.Bool("network-names"))
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err = showConns(data, c.String("delimiter"), c.Bool("network-names"))
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

const (
	day  = time.Minute * 60 * 24
	year = 365 * day
)

// https://gist.github.com/harshavardhana/327e0577c4fed9211f65#gistcomment-2557682
func duration(d time.Duration) string {
	if d < day {
		return d.String()
	}

	var b strings.Builder

	if d >= year {
		years := d / year
		fmt.Fprintf(&b, "%dy", years)
		d -= years * year
	}

	days := d / day
	d -= days * day
	fmt.Fprintf(&b, "%dd%s", days, d)

	return b.String()
}

func showConns(connResults []uconn.LongConnAnalysisView, delim string, showNetNames bool) error {

	var headerFields []string
	if showNetNames {
		headerFields = []string{"Source Network", "Destination Network", "Source IP", "Destination IP", "Port:Protocol:Service", "Duration"}
	} else {
		headerFields = []string{"Source IP", "Destination IP", "Port:Protocol:Service", "Duration"}
	}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headerFields, delim))
	for _, result := range connResults {
		var row []string
		if showNetNames {
			row = []string{
				result.SrcNetworkName,
				result.DstNetworkName,
				result.SrcIP,
				result.DstIP,
				strings.Join(result.Tuples, " "),
				f(result.MaxDuration),
			}
		} else {
			row = []string{
				result.SrcIP,
				result.DstIP,
				strings.Join(result.Tuples, " "),
				f(result.MaxDuration),
			}
		}

		fmt.Println(strings.Join(row, delim))
	}
	return nil
}

func showConnsHuman(connResults []uconn.LongConnAnalysisView, showNetNames bool) error {
	table := tablewriter.NewWriter(os.Stdout)

	var headerFields []string
	if showNetNames {
		headerFields = []string{"Source Network", "Destination Network", "Source IP", "Destination IP", "Port:Protocol:Service", "Duration"}
	} else {
		headerFields = []string{"Source IP", "Destination IP", "Port:Protocol:Service", "Duration"}
	}

	table.SetHeader(headerFields)
	for _, result := range connResults {
		var row []string
		if showNetNames {
			row = []string{
				result.SrcNetworkName,
				result.DstNetworkName,
				result.SrcIP,
				result.DstIP,
				strings.Join(result.Tuples, " "),
				duration(time.Duration(int(result.MaxDuration * float64(time.Second)))),
			}
		} else {
			row = []string{
				result.SrcIP,
				result.DstIP,
				strings.Join(result.Tuples, " "),
				duration(time.Duration(int(result.MaxDuration * float64(time.Second)))),
			}
		}

		table.Append(row)
	}
	table.Render()
	return nil
}

//getLongConnsResultsView gets the long connection results
func getLongConnsResultsView(res *resources.Resources, thresh int, sort string, sortDirection, limit int, noLimit bool) ([]uconn.LongConnAnalysisView, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var longConnResults []uconn.LongConnAnalysisView

	longConnQuery := []bson.M{
		bson.M{"$match": bson.M{"dat.maxdur": bson.M{"$gt": thresh}}},
		bson.M{"$project": bson.M{
			"src":              1,
			"src_network_uuid": 1,
			"src_network_name": 1,
			"dst":              1,
			"dst_network_uuid": 1,
			"dst_network_name": 1,
			"maxdur":           "$dat.maxdur",
			"tuples":           bson.M{"$ifNull": []interface{}{"$dat.tuples", []interface{}{}}},
		}},
		bson.M{"$unwind": "$maxdur"},
		bson.M{"$unwind": "$tuples"},
		bson.M{"$unwind": "$tuples"}, // not an error, must be done twice
		bson.M{"$group": bson.M{
			"_id":              "$_id",
			"maxdur":           bson.M{"$max": "$maxdur"},
			"src":              bson.M{"$first": "$src"},
			"src_network_uuid": bson.M{"$first": "$src_network_uuid"},
			"src_network_name": bson.M{"$first": "$src_network_name"},
			"dst":              bson.M{"$first": "$dst"},
			"dst_network_uuid": bson.M{"$first": "$dst_network_uuid"},
			"dst_network_name": bson.M{"$first": "$dst_network_name"},
			"tuples":           bson.M{"$addToSet": "$tuples"},
		}},
		bson.M{"$project": bson.M{
			"maxdur":           1,
			"src":              1,
			"src_network_uuid": 1,
			"src_network_name": 1,
			"dst":              1,
			"dst_network_uuid": 1,
			"dst_network_name": 1,
			"tuples":           bson.M{"$slice": []interface{}{"$tuples", 5}},
		}},
		bson.M{"$sort": bson.M{sort: sortDirection}},
	}

	if !noLimit {
		longConnQuery = append(longConnQuery, bson.M{"$limit": limit})
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).Pipe(longConnQuery).AllowDiskUse().All(&longConnResults)

	return longConnResults, err

}
