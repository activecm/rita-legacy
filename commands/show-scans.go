package commands

import (
	"encoding/csv"
	"os"
	"strconv"

	"github.com/activecm/rita/datatypes/scanning"
	"github.com/activecm/rita/resources"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:      "show-scans",
		Usage:     "Print scanning information",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
			configFlag,
			cli.BoolFlag{
				Name:  "ports, P",
				Usage: "Show which individual ports were scanned",
			},
		},
		Action: func(c *cli.Context) error {
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}
			showPorts := c.Bool("ports")

			res := resources.InitResources(c.String("config"))

			var scans []scanning.Scan
			coll := res.DB.Session.DB(db).C(res.Config.T.Scanning.ScanTable)
			coll.Find(nil).All(&scans)

			if len(scans) == 0 {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showScansHuman(scans, showPorts)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err := showScans(scans, showPorts)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showScans(scans []scanning.Scan, showPorts bool) error {
	csvWriter := csv.NewWriter(os.Stdout)
	header := []string{"Source", "Destination", "Ports Scanned"}
	if showPorts {
		header = append(header, "Ports")
	}
	csvWriter.Write(header)
	for _, scan := range scans {
		data := []string{scan.Src, scan.Dst, strconv.Itoa(scan.PortCount)}

		if showPorts {
			portSet := make([]byte, scan.PortCount*3)
			for i, port := range scan.PortSet {
				if i != 0 {
					strconv.AppendQuote(portSet, " ")
				}
				strconv.AppendInt(portSet, int64(port), 10)
			}
			data = append(data, string(portSet))
		}

		csvWriter.Write(data)
	}
	csvWriter.Flush()
	return nil
}

// showScans prints all scans for a given database
func showScansHuman(scans []scanning.Scan, showPorts bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Source", "Destination", "Ports Scanned"}
	if showPorts {
		header = append(header, "Ports")
	}
	table.SetHeader(header)
	for _, scan := range scans {
		data := []string{scan.Src, scan.Dst, strconv.Itoa(scan.PortCount)}

		if showPorts {
			portSet := make([]byte, scan.PortCount*3)
			for i, port := range scan.PortSet {
				if i != 0 {
					strconv.AppendQuote(portSet, " ")
				}
				strconv.AppendInt(portSet, int64(port), 10)
			}
			data = append(data, string(portSet))
		}

		table.Append(data)
	}
	table.Render()
	return nil
}
