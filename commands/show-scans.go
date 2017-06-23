package commands

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"text/template"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/scanning"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:  "show-scans",
		Usage: "Print scanning information",
		Flags: []cli.Flag{
			humanFlag,
			databaseFlag,
			configFlag,
		},
		Action: func(c *cli.Context) error {
			if c.String("database") == "" {
				return cli.NewExitError("Specify a database with -d", -1)
			}

			res := database.InitResources(c.String("config"))

			var scans []scanning.Scan
			coll := res.DB.Session.DB(c.String("database")).C(res.System.ScanningConfig.ScanTable)
			coll.Find(nil).All(&scans)

			if c.Bool("human-readable") {
				return showScansHuman(scans)
			}
			return showScans(scans)
		},
	}
	bootstrapCommands(command)
}

func showScans(scans []scanning.Scan) error {
	tmpl := "{{.Src}},{{.Dst}},{{.PortCount}},{{range $idx, $port := .PortSet}}{{if $idx}} {{end}}{{ $port }}{{end}}\r\n"

	out, err := template.New("scn").Parse(tmpl)
	if err != nil {
		return err
	}

	var error error
	for _, scan := range scans {
		sort.Ints(scan.PortSet)
		err := out.Execute(os.Stdout, scan)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
			error = err
		}
	}
	return error
}

// showScans prints all scans for a given database
func showScansHuman(scans []scanning.Scan) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Source", "Destination", "Ports Scanned"})
	for _, scan := range scans {
		table.Append([]string{scan.Src, scan.Dst, strconv.Itoa(scan.PortCount)})
	}
	table.Render()
	return nil
}
