package printing

import (
	"bytes"
	"html/template"
	"os"
	"strconv"

	"github.com/bglebrun/rita/analysis/beacon"
	"github.com/bglebrun/rita/database"
	beaconData "github.com/bglebrun/rita/datatypes/beacon"
	"github.com/bglebrun/rita/printing/templates"
	"github.com/olekukonko/tablewriter"
)

func printBeacons(db string, dir string, res *database.Resources) error {
	res.DB.SelectDB(db)
	var data []beaconData.BeaconAnalysisView
	ssn := res.DB.Session.Copy()
	beacon.GetBeaconResultsView(res, ssn, 0).All(&data)
	ssn.Close()

	return printBeaconHTML(dir, db, data)
}

func printBeaconHTML(dir string, db string, data []beaconData.BeaconAnalysisView) error {
	f, err := os.Create(dir + "beacons.html")
	if err != nil {
		return err
	}
	defer f.Close()

	w := new(bytes.Buffer)

	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Score", "Source IP", "Destination IP",
		"Connections", "Avg. Bytes", "Intvl Range", "Top Intvl",
		"Top Intvl Count", "Intvl Skew", "Intvl Dispersion", "Intvl Duration"})
	fl := func(fl float64) string {
		return strconv.FormatFloat(fl, 'g', 6, 64)
	}
	in := func(in int64) string {
		return strconv.FormatInt(in, 10)
	}
	for _, d := range data {
		table.Append(
			[]string{
				fl(d.TS_score), d.Src, d.Dst, in(d.Connections), fl(d.AvgBytes),
				in(d.TS_iRange), in(d.TS_iMode), in(d.TS_iModeCount), fl(d.TS_iSkew),
				in(d.TS_iDispersion), fl(d.TS_duration)})
	}
	table.Render()
	out, err := template.New("beacon.html").Parse(templates.BeaconsTempl)
	if err != nil {
		return err
	}
	return out.Execute(f, &scan{Dbs: db, Writer: w.String()})
}
