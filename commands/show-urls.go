package commands

import (
	"encoding/csv"
	"os"
	"strings"

	"github.com/activecm/rita/datatypes/urls"
	"github.com/activecm/rita/resources"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	longURLs := cli.Command{

		Name:      "show-long-urls",
		Usage:     "Print the longest urls",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
			configFlag,
		},
		Action: func(c *cli.Context) error {
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}

			res := resources.InitResources(c.String("config"))

			var urls []urls.URL
			coll := res.DB.Session.DB(db).C(res.Config.T.Urls.UrlsTable)

			coll.Find(nil).Sort("-length").All(&urls)

			if len(urls) == 0 {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showURLsHuman(urls)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}

			err := showURLs(urls)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}

	vistedURLs := cli.Command{

		Name:      "show-most-visited-urls",
		Usage:     "Print the most visited urls",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
		},
		Action: func(c *cli.Context) error {
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}

			res := resources.InitResources(c.String("config"))

			var urls []urls.URL
			coll := res.DB.Session.DB(db).C(res.Config.T.Urls.UrlsTable)

			coll.Find(nil).Sort("-count").All(&urls)

			if len(urls) == 0 {
				return cli.NewExitError("No results were found for "+db, -1)
			}

			if c.Bool("human-readable") {
				err := showURLsHuman(urls)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				return nil
			}
			err := showURLs(urls)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(longURLs, vistedURLs)
}

func showURLs(urls []urls.URL) error {
	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Write([]string{"URL", "URI", "Length", "Times Visted", "IPs"})
	for _, url := range urls {
		csvWriter.Write([]string{
			url.URL, url.URI, i(url.Length), i(url.Count), strings.Join(url.IPs, " "),
		})
	}
	csvWriter.Flush()
	return nil
}

func showURLsHuman(urls []urls.URL) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetColWidth(50)
	table.SetHeader([]string{"URL", "URI", "Length", "Times Visted", "IPs"})
	for _, url := range urls {
		if len(url.URL) > 50 {
			url.URL = url.URL[0:47] + "..."
		}
		if len(url.URI) > 50 {
			url.URI = url.URI[0:47] + "..."
		}
		table.Append([]string{
			url.URL, url.URI, i(url.Length), i(url.Count), strings.Join(url.IPs, " "),
		})
	}
	table.Render()
	return nil
}
