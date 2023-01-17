package commands

import (
	"fmt"

	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/urfave/cli"
)

func init() {
	clean := cli.Command{
		Name:    "clean",
		Aliases: []string{"clean-databases"},
		Usage:   "Finds and removes broken databases. Prompts before deleting each database unless --force is provided.",
		Flags: []cli.Flag{
			ConfigFlag,
			forceFlag,
		},
		Action: cleanDatabase,
	}

	bootstrapCommands(clean)
}

// cleanDatabase finds and removes broken databases created by RITA
func cleanDatabase(c *cli.Context) error {
	res := resources.InitResources(getConfigFilePath(c))

	force := c.Bool("force")

	ritaAnalysisCollNames := map[string]string{
		res.Config.T.Structure.UniqueConnTable:      "Unique Connection Analysis",
		res.Config.T.Structure.HostTable:            "Host Analysis",
		res.Config.T.DNS.HostnamesTable:             "Hostnames Analysis",
		res.Config.T.DNS.ExplodedDNSTable:           "ExplodedDNS Analysis",
		res.Config.T.Structure.UniqueConnProxyTable: "Uconn Proxy Analysis",
		res.Config.T.BeaconProxy.BeaconProxyTable:   "Proxy Beacon Analysis",
		res.Config.T.Beacon.BeaconTable:             "Beacon Analysis",
		res.Config.T.Structure.SNIConnTable:         "SNI Beacon Analysis",
		res.Config.T.BeaconSNI.BeaconSNITable:       "SNI Connection Analysis",
		res.Config.T.UserAgent.UserAgentTable:       "UserAgent Analysis",
		res.Config.T.Cert.CertificateTable:          "Certificate Analysis",
	}

	session := res.DB.Session.Copy()

	allDBs, err := session.DatabaseNames()

	if err != nil {
		fmt.Print("Clean failed: Failed to fetch database list.\n")
		return cli.NewExitError(err.Error(), -1)
	}

	// dbsWithRitaColls maps a dataset to the list of RITA collections it contains
	dbsWithRitaColls := make(map[string][]string)

	for _, db := range allDBs {
		existsInMeta, err := res.MetaDB.DBExists(db)
		if err != nil {
			fmt.Print("Clean failed: Failed to query MetaDB.\n")
			return cli.NewExitError(err.Error(), -1)
		}

		if existsInMeta { // only select datasets without MetaDB records
			continue
		}

		collections, err := session.DB(db).CollectionNames()
		if err != nil {
			fmt.Print("Clean failed: Failed to fetch collection list.\n")
			return cli.NewExitError(err.Error(), -1)
		}

		var ritaCollsFound []string

		for _, collection := range collections {
			for ritaColl := range ritaAnalysisCollNames {
				if collection == ritaColl {
					ritaCollsFound = append(ritaCollsFound, ritaColl)
				}
			}
		}
		if len(ritaCollsFound) > 0 {
			dbsWithRitaColls[db] = ritaCollsFound
		}
	}

	if len(dbsWithRitaColls) == 0 {
		fmt.Print("Clean successful: Nothing to remove.\n")
		return nil
	}

	fmt.Print("Deleting the following broken RITA datasets:\n")

	var removedDBs []string
	var removedTotalSize int64

	for matchingDB, matchingColls := range dbsWithRitaColls {
		var dbSize struct {
			DataSize  int64 `bson:"dataSize"`
			IndexSize int64 `bson:"indexSize"`
		}

		err = session.DB(matchingDB).Run(bson.D{
			{Name: "dbStats", Value: 1},
			{Name: "scale", Value: 1024 * 1024},
		}, &dbSize)

		if err != nil {
			fmt.Print("Clean failed: Failed to gather size of dataset\n")
			return cli.NewExitError(err.Error(), -1)
		}

		fmt.Printf("\t[-] %s takes up %d MB on disk\n", matchingDB, dbSize.DataSize+dbSize.IndexSize)
		fmt.Printf("\t[-] %s contains results from the following analyses:\n", matchingDB)
		for _, matchingColl := range matchingColls {
			fmt.Printf("\t\t[-] %s\n", ritaAnalysisCollNames[matchingColl])
		}

		if force || confirmAction("\t [?] Confirm we'll be deleting "+matchingDB) {
			err = session.DB(matchingDB).DropDatabase()
			if err != nil {
				fmt.Print("Clean failed: Failed to delete dataset\n")
				return cli.NewExitError(err.Error(), -1)
			}

			removedDBs = append(removedDBs, matchingDB)
			removedTotalSize += dbSize.DataSize + dbSize.IndexSize
		}
		fmt.Println()
	}

	fmt.Printf("Clean successful: Removed %d dataset(s) totalling %d MB on disk.\n", len(removedDBs), removedTotalSize)
	return nil
}
