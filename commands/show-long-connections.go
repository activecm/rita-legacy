package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:      "show-long-connections",
		Usage:     "Print long connections and relevant information",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			ConfigFlag,
			humanFlag,
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

			res := resources.InitResources(getConfigFilePath(c))
			res.DB.SelectDB(db)

			thresh := 60 // 1 minute
			data, err := uconn.LongConnResults(res, thresh, c.Int("limit"), c.Bool("no-limit"))

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

func showConns(connResults []uconn.LongConnResult, delim string, showNetNames bool) error {

	var headerFields []string
	if showNetNames {
		headerFields = []string{"Source Network", "Destination Network", "Source IP", "Destination IP", "Port:Protocol:Service", "Duration", "State"}
	} else {
		headerFields = []string{"Source IP", "Destination IP", "Port:Protocol:Service", "Duration", "State"}
	}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headerFields, delim))
	for _, result := range connResults {
		var row []string

		// Convert the true/false open/closed state to a nice string
		state := "closed"
		if result.Open {
			state = "open"
		}

		if showNetNames {
			row = []string{
				result.SrcNetworkName,
				result.DstNetworkName,
				result.SrcIP,
				result.DstIP,
				strings.Join(result.Tuples, " "),
				f(result.MaxDuration),
				state,
			}
		} else {
			row = []string{
				result.SrcIP,
				result.DstIP,
				strings.Join(result.Tuples, " "),
				f(result.MaxDuration),
				state,
			}
		}

		fmt.Println(strings.Join(row, delim))
	}
	return nil
}

func showConnsHuman(connResults []uconn.LongConnResult, showNetNames bool) error {
	table := tablewriter.NewWriter(os.Stdout)

	var headerFields []string
	if showNetNames {
		headerFields = []string{"Source Network", "Destination Network", "Source IP", "Destination IP", "Port:Protocol:Service", "Duration", "State"}
	} else {
		headerFields = []string{"Source IP", "Destination IP", "Port:Protocol:Service", "Duration", "State"}
	}

	table.SetHeader(headerFields)
	for _, result := range connResults {
		var row []string

		// Convert the true/false open/closed state to a nice string
		state := "closed"
		if result.Open {
			state = "open"
		}

		if showNetNames {
			row = []string{
				result.SrcNetworkName,
				result.DstNetworkName,
				result.SrcIP,
				result.DstIP,
				strings.Join(result.Tuples, " "),
				util.FormatDuration(time.Duration(int(result.MaxDuration * float64(time.Second)))),
				state,
			}
		} else {
			row = []string{
				result.SrcIP,
				result.DstIP,
				strings.Join(result.Tuples, " "),
				util.FormatDuration(time.Duration(int(result.MaxDuration * float64(time.Second)))),
				state,
			}
		}

		table.Append(row)
	}
	table.Render()
	return nil
}
