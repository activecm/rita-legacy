package commands

import (
	"encoding/csv"
	"os"

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

	if c.Bool("human-readable") {
		err := showBeaconReport(data)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
		return nil
	}

	err = showBeaconCsv(data)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}
	return nil
}

func showBeaconReport(data []beacon.AnalysisView) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Score", "Source IP", "Destination IP",
		"Connections", "Avg. Bytes", "Intvl Range", "Size Range", "Top Intvl",
		"Top Size", "Top Intvl Count", "Top Size Count", "Intvl Skew",
		"Size Skew", "Intvl Dispersion", "Size Dispersion"})

	for _, d := range data {
		table.Append(
			[]string{
				f(d.Score), d.Src, d.Dst, i(d.Connections), f(d.AvgBytes),
				i(d.Ts.Range), i(d.Ds.Range), i(d.Ts.Mode), i(d.Ds.Mode),
				i(d.Ts.ModeCount), i(d.Ds.ModeCount), f(d.Ts.Skew), f(d.Ds.Skew),
				i(d.Ts.Dispersion), i(d.Ds.Dispersion),
			},
		)
	}
	table.Render()
	return nil
}

func showBeaconCsv(data []beacon.AnalysisView) error {
	csvWriter := csv.NewWriter(os.Stdout)
	headers := []string{"Score", "Source IP", "Destination IP",
		"Connections", "Avg Bytes", "Intvl Range", "Size Range", "Top Intvl",
		"Top Size", "Top Intvl Count", "Top Size Count", "Intvl Skew",
		"Size Skew", "Intvl Dispersion", "Size Dispersion"}
	csvWriter.Write(headers)

	for _, d := range data {
		csvWriter.Write(
			[]string{
				f(d.Score), d.Src, d.Dst, i(d.Connections), f(d.AvgBytes),
				i(d.Ts.Range), i(d.Ds.Range), i(d.Ts.Mode), i(d.Ds.Mode),
				i(d.Ts.ModeCount), i(d.Ds.ModeCount), f(d.Ts.Skew), f(d.Ds.Skew),
				i(d.Ts.Dispersion), i(d.Ds.Dispersion),
			},
		)
	}
	csvWriter.Flush()
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
