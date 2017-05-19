package printing

import (
	"bytes"
	"html/template"
	"os"
	"strconv"
	"strings"

	"github.com/bglebrun/rita/analysis/blacklisted"
	"github.com/bglebrun/rita/database"
	blacklistedData "github.com/bglebrun/rita/datatypes/blacklisted"
	htmlTempl "github.com/bglebrun/rita/printing/templates"
	"github.com/olekukonko/tablewriter"
)

func printBlacklisted(db string, res *database.Resources) error {
	res.DB.SelectDB(db)

	var result blacklistedData.Blacklist
	var results []blacklistedData.Blacklist

	coll := res.DB.Session.DB(db).C(res.System.BlacklistedConfig.BlacklistTable)
	iter := coll.Find(nil).Sort("-count").Iter()

	for iter.Next(&result) {
		blacklisted.SetBlacklistSources(res, &result)
		results = append(results, result)
	}

	return printBlacklistedHTML(results, db)
}

// TODO: Convert this over to tablewriter
// printBlacklistedHTML prints all blacklisted for a given database
func printBlacklistedHTML(results []blacklistedData.Blacklist, db string) error {
	f, err := os.Create("blacklisted.html")
	if err != nil {
		return err
	}
	defer f.Close()

	w := new(bytes.Buffer)

	table := tablewriter.NewWriter(w)
	table.SetColWidth(100)
	table.SetHeader([]string{"Host", "Score", "Sources"})
	for _, result := range results {
		table.Append([]string{
			result.Host, strconv.Itoa(result.Score), strings.Join(result.Sources, ", "),
		})
	}

	table.Render()
	out, err := template.New("blacklisted.html").Parse(htmlTempl.BlacklistedTempl)
	if err != nil {
		return err
	}

	return out.Execute(f, &scan{Dbs: db, Writer: w.String()})
}
