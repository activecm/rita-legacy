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
			allFlag,
		},
		Action: showBeacons,
	}

	bootstrapCommands(command)
}

func showBeacons(c *cli.Context) error {
	if c.String("database") == "" {
		return cli.NewExitError("Specify a database with -d", -1)
	}
	res := database.InitResources("")
	res.DB.SelectDB(c.String("database"))

	var data []beaconData.BeaconAnalysisView
	cutoffScore := .7
	if c.Bool("all") {
		cutoffScore = 0
	}

	ssn := res.DB.Session.Copy()
	beacon.GetBeaconResultsView(res, ssn, cutoffScore).All(&data)
	ssn.Close()

	if c.Bool("human-readable") {
		return showBeaconReport(data)
	}

	return showBeaconCsv(data)
}

func showBeaconReport(data []beaconData.BeaconAnalysisView) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Score", "Source IP", "Destination IP",
		"Connections", "Avg. Bytes", "Intvl Range", "Top Intvl",
		"Top Intvl Count", "Intvl Skew", "Intvl Dispersion", "Intvl Duration"})
	f := func(f float64) string {
		return strconv.FormatFloat(f, 'g', 6, 64)
	}
	i := func(i int64) string {
		return strconv.FormatInt(i, 10)
	}
	for _, d := range data {
		table.Append(
			[]string{
				f(d.TS_score), d.Src, d.Dst, i(d.Connections), f(d.AvgBytes),
				i(d.TS_iRange), i(d.TS_iMode), i(d.TS_iModeCount), f(d.TS_iSkew),
				i(d.TS_iDispersion), f(d.TS_duration)})
	}
	table.Render()
	return nil
}

func showBeaconCsv(data []beaconData.BeaconAnalysisView) error {
	tmpl := "{{.TS_score}},{{.Src}},{{.Dst}},{{.Connections}},{{.AvgBytes}},"
	tmpl += "{{.TS_iRange}},{{.TS_iMode}},{{.TS_iModeCount}},"
	tmpl += "{{.TS_iSkew}},{{.TS_iDispersion}},{{.TS_duration}}\n"

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
