package commands

import (
	"fmt"
	"os"
	"text/template"

	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/TBD"
	"github.com/urfave/cli"
)

var cutoffScore float64

func init() {
	command := cli.Command{
		Name:  "show-beacons",
		Usage: "print beacon information to standard out",
		Flags: []cli.Flag{
			humanFlag,
			databaseFlag,
			cli.Float64Flag{
				Name:        "score, s",
				Usage:       "change the beacon cutoff score",
				Destination: &cutoffScore,
				Value:       .7,
			},
		},
		Action: showBeacons,
	}

	bootstrapCommands(command)
}

func showBeacons(c *cli.Context) error {
	if c.String("dataset") == "" {
		return cli.NewExitError("No dataset was not specified", -1)
	}
	conf := config.InitConfig("")
	conf.System.DB = c.String("dataset")

	db := database.NewDB(conf)
	data := db.GetTBDResultsView(cutoffScore)

	if humanreadable {
		return showBeacon2Report(data)
	}

	return showBeacon2Csv(data)
}

func showBeacon2Report(data []TBD.TBDAnalysisView) error {
	hdr := " Score |   Source IP    | Destination IP | Connections |"
	hdr += " Avg. Bytes | Intvl Range | Top Intvl | Top Intvl Count |"
	hdr += " Intvl Skew | Intvl Dispersion | Intvl Duration \n"
	tmpl := `{{.TS_score | printf "%7.3f"}}` + " "
	tmpl += `{{.Src | printf "%16s"}}` + " "
	tmpl += `{{.Dst | printf "%16s"}}` + " "
	tmpl += `{{.Connections | printf "%13d"}}` + " "
	tmpl += `{{.AvgBytes | printf "%12.3f"}}` + " "
	tmpl += `{{.TS_iRange | printf "%13d"}}` + " "
	tmpl += `{{.TS_iMode | printf "%11d"}}` + " "
	tmpl += `{{.TS_iModeCount | printf "%17d"}}` + " "
	tmpl += `{{.TS_iSkew | printf "%12.3f"}}` + " "
	tmpl += `{{.TS_iDispersion | printf "%18d"}}` + " "
	tmpl += `{{.TS_duration | printf "%16.3f"}}` + "\n"

	out, err := template.New("tbd").Parse(tmpl)
	if err != nil {
		return err
	}
	fmt.Print(hdr)
	for _, d := range data {
		err := out.Execute(os.Stdout, d)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}
	return nil
}

func showBeacon2Csv(data []TBD.TBDAnalysisView) error {
	tmpl := "{{.TS_score}},{{.Src}},{{.Dst}},{{.Connections}},{{.AvgBytes}},"
	tmpl += "{{.TS_iRange}},{{.TS_iMode}},{{.TS_iModeCount}},"
	tmpl += "{{.TS_iSkew}},{{.TS_iDispersion}},{{.TS_duration}}\n"

	out, err := template.New("tbd").Parse(tmpl)
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
