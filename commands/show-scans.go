package commands

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"text/template"

	"github.com/ocmdev/rita/config"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:  "show-scans",
		Usage: "print scanning information to standard out",
		Flags: []cli.Flag{
			humanFlag,
			databaseFlag,
		},
		Action: func(c *cli.Context) error {
			if c.String("dataset") == "" {
				return errors.New("No dataset specified")
			}
			if humanreadable {
				return showScansHuman(c)
			}
			return showScans(c)
		},
	}
	bootstrapCommands(command)
}

type scanres struct {
	Src   string `bson:"src"`
	Dst   string `bson:"dst"`
	Count int    `bson:"port_count"`
	Ports []int  `bson:"port_set"`
	Range string
}

//TODO: implement sorting
type scanresset []scanres

// implement the sort.Interface
func (slice scanresset) Len() int {
	return len(slice)
}

func (slice scanresset) Less(i, j int) bool {
	return slice[i].Count > slice[j].Count
}

func (slice scanresset) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func showScans(c *cli.Context) error {
	tmpl := "{{.Src}},{{.Dst}},{{.Count}},[{{range $idx, $port := .Ports}}{{ $port }};{{end}}]\r\n"

	out, err := template.New("scn").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")

	var res scanres
	coll := conf.Session.DB(c.String("dataset")).C(conf.System.ScanningConfig.ScanTable)
	iter := coll.Find(nil).Iter()

	for iter.Next(&res) {
		err := out.Execute(os.Stdout, res)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}
	return nil
}

// showScans prints all scans for a given database
func showScansHuman(c *cli.Context) error {

	cols := "           source\t            dest\tport-count\tRange\n"
	cols += "-----------------\t----------------\t----------\t-----\n"
	tmpl := "{{.Src}}\t{{.Dst}}\t" + `{{.Count | printf "%10d"}}` + "\t{{.Range}}\n"
	out, err := template.New("scn").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")

	var res scanres
	coll := conf.Session.DB(c.String("dataset")).C(conf.System.ScanningConfig.ScanTable)
	iter := coll.Find(nil).Iter()

	fmt.Printf(cols)
	for iter.Next(&res) {
		res.Src = padAddr(res.Src)
		res.Dst = padAddr(res.Dst)
		res.Range = getPortRange(res)
		err := out.Execute(os.Stdout, res)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}

	return nil
}

// getPortRange takes in a scanning result structure and returns a representation
// of the range of that particular scan
func getPortRange(r scanres) string {
	biggest := -1
	smallest := 999999
	for _, port := range r.Ports {
		if port > biggest {
			biggest = port
		}
		if port < smallest {
			smallest = port
		}
	}

	return strconv.Itoa(smallest) + "--" + strconv.Itoa(biggest)
}
