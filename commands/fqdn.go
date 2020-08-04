package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

// FQDN returns a clipped list of associated hostnames and a total count
type FqdnResult struct {
	Hostname string `bson:"_id"`
}

func init() {

	databases := cli.Command{
		Name:    "fqdn",
		Aliases: []string{"get-hostnames"},
		Usage:   "retrieves a list hostnames associated with an ip",
		Flags: []cli.Flag{
			humanFlag,
			configFlag,
			delimFlag,
		},
		ArgsUsage: "<database> <ip>",
		Action:    printFQDN,
	}

	bootstrapCommands(databases)
}

func printFQDN(c *cli.Context) error {
	// get database name
	db := c.Args().Get(0)

	if db == "" {
		return cli.NewExitError("Specify a database", -1)
	}
	// get ip
	ip := c.Args().Get(1)

	if ip == "" {
		return cli.NewExitError("Specify an ip", -1)
	}

	// check for regex
	ip = checkRegEx(ip)

	// setting up to use database stuff (aka don't worry about it)
	res := resources.InitResources(c.String("config"))
	res.DB.SelectDB(db)

	// get that data!
	data, err := getFqdnResult(res, ip, 1000)

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if len(data) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	// print pretty
	if c.Bool("human-readable") {
		err := showFQDNHuman(data, c.String("delimiter"))
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
		return nil
	}

	// just print
	err = showFQDN(data, c.String("delimiter"))
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}

	return nil
}

func showFQDN(data []FqdnResult, delim string) error {
	headers := []string{"Hostnames"}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headers, delim))
	for _, d := range data {
		fmt.Println(
			strings.Join(
				[]string{
					d.Hostname,
				},
				delim,
			),
		)
	}
	return nil
}

func showFQDNHuman(data []FqdnResult, delim string) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Hostnames"})

	for _, d := range data {
		table.Append(
			[]string{d.Hostname},
		)
	}
	table.Render()
	return nil
}

//getFqdnResult ...
func getFqdnResult(res *resources.Resources, ip string, limit int) ([]FqdnResult, error) {

	// setting up to query mongo stuff (aka don't worry about it)
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	// declare your result variable
	var fqdn []FqdnResult

	fqdnQuery := []bson.M{
		bson.M{"$match": bson.M{
			"dat.ips": bson.RegEx{ip, ""},
		}},
		bson.M{"$group": bson.M{
			"_id": "$host",
		}},
		bson.M{"$limit": limit},
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C("hostnames").Pipe(fqdnQuery).AllowDiskUse().All(&fqdn)

	return fqdn, err

}

//checkRegEx ...
func checkRegEx(val string) string {
	// remove any surrounding slashes since they are the common syntax for the
	// mongo shell, making it possible someone will try to use them
	// (they will not work for the GO regex query)
	val = strings.Replace(val, "/", "", -1)
	// escape other characters
	val = strings.Replace(val, ".", "\\.", -1)

	length := len(val)

	if val[:1] == "*" {
		val = val[1:] + "$"
	} else if val[length-1:] == "*" {
		val = "^" + val[:length-1]
	} else {
		val = "^" + val
	}
	return val
}
