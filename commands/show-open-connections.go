package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/resources"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:      "show-open-connections",
		Usage:     "Print open connections and relevant information",
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
			data, err := uconn.OpenConnResults(res, thresh, c.Int("limit"), c.Bool("no-limit"))

			if err != nil {
				res.Log.Error(err)
				return cli.NewExitError(err, -1)
			}

			if !(len(data) > 0) {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showOpenConnsHuman(data, c.Bool("network-names"))
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err = showOpenConns(data, c.String("delimiter"), c.Bool("network-names"))
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}

// https://gist.github.com/harshavardhana/327e0577c4fed9211f65#gistcomment-2557682
func openDuration(d time.Duration) string {

	const (
		day  = time.Minute * 60 * 24
		year = 365 * day
	)

	if d < day {
		return d.String()
	}

	var b strings.Builder

	if d >= year {
		years := d / year
		fmt.Fprintf(&b, "%dy", years)
		d -= years * year
	}

	days := d / day
	d -= days * day
	fmt.Fprintf(&b, "%dd%s", days, d)

	return b.String()
}

func showOpenConns(connResults []uconn.OpenConnResult, delim string, showNetNames bool) error {

	var headerFields []string
	if showNetNames {
		headerFields = []string{"Source Network", "Destination Network", "Source IP", "Destination IP", "Port:Protocol:Service", "Duration", "Bytes", "Zeek UID"}
	} else {
		headerFields = []string{"Source IP", "Destination IP", "Port:Protocol:Service", "Duration", "Bytes", "Zeek UID"}
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
				result.Tuple,
				openDuration(time.Duration(int(result.Duration * float64(time.Second)))),
				strconv.Itoa(result.Bytes),
				result.UID,
			}
		} else {
			row = []string{
				result.SrcIP,
				result.DstIP,
				result.Tuple,
				openDuration(time.Duration(int(result.Duration * float64(time.Second)))),
				strconv.Itoa(result.Bytes),
				result.UID,
			}
		}

		fmt.Println(strings.Join(row, delim))
	}
	return nil
}

func showOpenConnsHuman(connResults []uconn.OpenConnResult, showNetNames bool) error {
	table := tablewriter.NewWriter(os.Stdout)

	var headerFields []string
	if showNetNames {
		headerFields = []string{"Source Network", "Destination Network", "Source IP", "Destination IP", "Port:Protocol:Service", "Duration", "Bytes", "Zeek UID"}
	} else {
		headerFields = []string{"Source IP", "Destination IP", "Port:Protocol:Service", "Duration", "Bytes", "Zeek UID"}
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
				result.Tuple,
				openDuration(time.Duration(int(result.Duration * float64(time.Second)))),
				strconv.Itoa(result.Bytes),
				result.UID,
			}
		} else {
			row = []string{
				result.SrcIP,
				result.DstIP,
				result.Tuple,
				openDuration(time.Duration(int(result.Duration * float64(time.Second)))),
				strconv.Itoa(result.Bytes),
				result.UID,
			}
		}

		table.Append(row)
	}
	table.Render()
	return nil
}
