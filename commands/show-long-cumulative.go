package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/resources"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:      "show-long-cumulative",
		Usage:     "Print long cumulative connections and relevant information",
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

			data, err := uconn.LongConnCumulativeResults(res)

			if err != nil {
				res.Log.Error(err)
				return cli.NewExitError(err, -1)
			}

			if !(len(data) > 0) {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showConnsHumanCumulative(data, c.Bool("network-names"))
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err = showConnsCumulative(data, c.String("delimiter"), c.Bool("network-names"))
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

func showConnsCumulative(connResults []uconn.LongConnResult, delim string, showNetNames bool) error {

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

func showConnsHumanCumulative(connResults []uconn.LongConnResult, showNetNames bool) error {
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
