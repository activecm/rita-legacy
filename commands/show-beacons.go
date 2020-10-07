package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/activecm/rita/pkg/beacon"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:      "show-beacons",
		Usage:     "Print hosts which show signs of C2 software",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
			configFlag,
			delimFlag,
			netNamesFlag,
		},
		Action: showBeacons,
	}

	bootstrapCommands(command)
}

func showBeacons(c *cli.Context) error {
	db := c.Args().Get(0)
	if db == "" {
		return cli.NewExitError("Specify a database", -1)
	}
	res := resources.InitResources(c.String("config"))
	res.DB.SelectDB(db)

	data, err := getBeaconResultsView(res, 0)

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if !(len(data) > 0) {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	showNetNames := c.Bool("network-names")

	if c.Bool("human-readable") {
		err := showBeaconsHuman(data, showNetNames)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
		return nil
	}

	err = showBeaconsDelim(data, c.String("delimiter"), showNetNames)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}
	return nil
}

func showBeaconsHuman(data []beacon.AnalysisView, showNetNames bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	var headerFields []string
	if showNetNames {
		headerFields = []string{
			"Score", "Source Network", "Destination Network", "Source IP", "Destination IP",
			"Connections", "Avg. Bytes", "Intvl Range", "Size Range", "Top Intvl",
			"Top Size", "Top Intvl Count", "Top Size Count", "Intvl Skew",
			"Size Skew", "Intvl Dispersion", "Size Dispersion",
		}
	} else {
		headerFields = []string{
			"Score", "Source IP", "Destination IP",
			"Connections", "Avg. Bytes", "Intvl Range", "Size Range", "Top Intvl",
			"Top Size", "Top Intvl Count", "Top Size Count", "Intvl Skew",
			"Size Skew", "Intvl Dispersion", "Size Dispersion",
		}
	}

	table.SetHeader(headerFields)

	for _, d := range data {
		var row []string
		if showNetNames {
			row = []string{
				f(d.Score), d.SrcNetworkName, d.DstNetworkName,
				d.SrcIP, d.DstIP, i(d.Connections), f(d.AvgBytes),
				i(d.Ts.Range), i(d.Ds.Range), i(d.Ts.Mode), i(d.Ds.Mode),
				i(d.Ts.ModeCount), i(d.Ds.ModeCount), f(d.Ts.Skew), f(d.Ds.Skew),
				i(d.Ts.Dispersion), i(d.Ds.Dispersion),
			}
		} else {
			row = []string{
				f(d.Score), d.SrcIP, d.DstIP, i(d.Connections), f(d.AvgBytes),
				i(d.Ts.Range), i(d.Ds.Range), i(d.Ts.Mode), i(d.Ds.Mode),
				i(d.Ts.ModeCount), i(d.Ds.ModeCount), f(d.Ts.Skew), f(d.Ds.Skew),
				i(d.Ts.Dispersion), i(d.Ds.Dispersion),
			}
		}
		table.Append(row)
	}
	table.Render()
	return nil
}

func showBeaconsDelim(data []beacon.AnalysisView, delim string, showNetNames bool) error {
	var headerFields []string
	if showNetNames {
		headerFields = []string{
			"Score", "Source Network", "Destination Network", "Source IP", "Destination IP",
			"Connections", "Avg. Bytes", "Intvl Range", "Size Range", "Top Intvl",
			"Top Size", "Top Intvl Count", "Top Size Count", "Intvl Skew",
			"Size Skew", "Intvl Dispersion", "Size Dispersion",
		}
	} else {
		headerFields = []string{
			"Score", "Source IP", "Destination IP",
			"Connections", "Avg. Bytes", "Intvl Range", "Size Range", "Top Intvl",
			"Top Size", "Top Intvl Count", "Top Size Count", "Intvl Skew",
			"Size Skew", "Intvl Dispersion", "Size Dispersion",
		}
	}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headerFields, delim))
	for _, d := range data {

		var row []string
		if showNetNames {
			row = []string{
				f(d.Score), d.SrcNetworkName, d.DstNetworkName,
				d.SrcIP, d.DstIP, i(d.Connections), f(d.AvgBytes),
				i(d.Ts.Range), i(d.Ds.Range), i(d.Ts.Mode), i(d.Ds.Mode),
				i(d.Ts.ModeCount), i(d.Ds.ModeCount), f(d.Ts.Skew), f(d.Ds.Skew),
				i(d.Ts.Dispersion), i(d.Ds.Dispersion),
			}
		} else {
			row = []string{
				f(d.Score), d.SrcIP, d.DstIP, i(d.Connections), f(d.AvgBytes),
				i(d.Ts.Range), i(d.Ds.Range), i(d.Ts.Mode), i(d.Ds.Mode),
				i(d.Ts.ModeCount), i(d.Ds.ModeCount), f(d.Ts.Skew), f(d.Ds.Skew),
				i(d.Ts.Dispersion), i(d.Ds.Dispersion),
			}
		}

		fmt.Println(strings.Join(row, delim))
	}
	return nil
}

//getBeaconResultsView finds beacons greater than a given cutoffScore
//and links the data from the unique connections table back in to the results
func getBeaconResultsView(res *resources.Resources, cutoffScore float64) ([]beacon.AnalysisView, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var beacons []beacon.AnalysisView

	beaconQuery := bson.M{"score": bson.M{"$gt": cutoffScore}}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Beacon.BeaconTable).Find(beaconQuery).Sort("-score").All(&beacons)

	return beacons, err
}
