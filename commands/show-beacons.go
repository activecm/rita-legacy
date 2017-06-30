package commands

import (
	"fmt"
	"os"
	"strconv"
	"text/template"

	"github.com/ocmdev/rita/analysis/beacon"
	"github.com/ocmdev/rita/database"
	beaconData "github.com/ocmdev/rita/datatypes/beacon"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:  "show-beacons",
		Usage: "Print beacon information to standard out",
		Flags: []cli.Flag{
			humanFlag,
			databaseFlag,
			configFlag,
		},
		Action: showBeacons,
	}

	bootstrapCommands(command)
}

func showBeacons(c *cli.Context) error {
	if c.String("database") == "" {
		return cli.NewExitError("Specify a database with -d", -1)
	}
	res := database.InitResources(c.String("config"))
	res.DB.SelectDB(c.String("database"))

	var data []beaconData.BeaconAnalysisView

	ssn := res.DB.Session.Copy()
	resultsView := beacon.GetBeaconResultsView(res, ssn, 0)
	if resultsView == nil {
		return cli.NewExitError("No results were found for "+c.String("database"), -1)
	}
	resultsView.All(&data)
	ssn.Close()

	if c.Bool("human-readable") {
		err := showBeaconReport(data)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}

	err := showBeaconCsv(data)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}
	return nil
}

func showBeaconReport(data []beaconData.BeaconAnalysisView) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Score", "Source IP", "Destination IP",
		"Connections", "Avg. Bytes", "Intvl Range", "Size Range", "Top Intvl",
		"Top Size", "Top Intvl Count", "Top Size Count", "Intvl Skew",
		"Size Skew", "Intvl Dispersion", "Size Dispersion", "Intvl Duration"})

	f := func(f float64) string {
		return strconv.FormatFloat(f, 'g', 6, 64)
	}
	i := func(i int64) string {
		return strconv.FormatInt(i, 10)
	}
	for _, d := range data {
		table.Append(
			[]string{
				f(d.Score), d.Src, d.Dst, i(d.Connections), f(d.AvgBytes),
				i(d.TS_iRange), i(d.DS_range), i(d.TS_iMode), i(d.DS_mode),
				i(d.TS_iModeCount), i(d.DS_modeCount), f(d.TS_iSkew), f(d.DS_skew),
				i(d.TS_iDispersion), i(d.DS_dispersion), f(d.TS_duration)})
	}
	table.Render()
	return nil
}

func showBeaconCsv(data []beaconData.BeaconAnalysisView) error {
	tmpl := "{{.Score}},{{.Src}},{{.Dst}},{{.Connections}},{{.AvgBytes}},"
	tmpl += "{{.TS_iRange}},{{.DS_range}},{{.TS_iMode}},{{.DS_mode}},{{.TS_iModeCount}},"
	tmpl += "{{.DS_modeCount}},{{.TS_iSkew}},{{.DS_skew}},{{.TS_iDispersion}},"
	tmpl += "{{.DS_dispersion}},{{.TS_duration}}\n"

	out, err := template.New("beacon").Parse(tmpl)
	if err != nil {
		return err
	}
	for _, d := range data {
		err := out.Execute(os.Stdout, d)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}
	return nil
}
