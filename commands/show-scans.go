package commands

import (
	"errors"
	"fmt"
	"os"
	"text/template"

	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/datatypes/scanning"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:  "show-scans",
		Usage: "print scanning information to standard out",
		Flags: []cli.Flag{
			databaseFlag,
		},
		Action: func(c *cli.Context) error {
			if c.String("dataset") == "" {
				return errors.New("No dataset specified")
			}
			showScans(c.String("database"))
			return nil
		},
	}
	bootstrapCommands(command)
}

type scanres struct {
	Src   string `bson:"src"`
	Dst   string `bson:"dst"`
	Count int    `bson:"port_count"`
	Ports []int  `bson:"port_set"`
}

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

// showScans prints all scans for a given database
func showScans(dataset string) {

	cols := "source\tdest\tport-count\tports\n"
	tmpl := "{{.Src}}\t{{.Dst}}\t{{.PortSetCount}}\t"
	tmpl += "{{range $idx, $port := .PortSet}} {{ $port }} {{end}}\n"
	out, err := template.New("scn").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")
	conf.System.DB = dataset

	var res scanning.Scan

	coll := conf.Session.DB(dataset).C(conf.System.ScanningConfig.ScanTable)
	iter := coll.Find(nil).Iter()

	fmt.Printf(cols)
	for iter.Next(&res) {
		err := out.Execute(os.Stdout, res)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}

}
