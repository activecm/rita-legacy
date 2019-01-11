package conn

import(
	"github.com/activecm/rita/parser/parsetypes"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

func (fs *FSImporter) BulkDeleteSetup(conns []*Conn) (bulk, error) {
	resDB := fs.res.DB
	resConf := fs.res.Config
	logger := fs.res.Log

	var deleteQuery bson.M

	// open a new database session for the bulk deletion
	ssn := resDB.Session.Copy()
	defer ssn.Close()

	bulk := ssn.DB(targetDB).C(resConf.T.Structure.ConnTable).Bulk()
	bulk.Unordered()

	fmt.Println("\t[-] Removing unused connection info. This may take a while.")
	for _, uconn := range uconns {
		deleteQuery = bson.M{
			"$and": []bson.M{
				bson.M{"id_orig_h": uconn.src},
				bson.M{"id_resp_h": uconn.dst},
			}}
		bulk.RemoveAll(deleteQuery)

		// remove entry out of uconns map so it doesn't end up in uconns collection
		srcDst := uconn.src + uconn.dst
		delete(uconnMap, srcDst)
	}
}

func BulkDeleteRun(bulk *bulk) error {
	// Execute the bulk deletion
	bulkResult, err := bulk.Run()
	if err != nil {
		logger.WithFields(log.Fields{
			"bulkResult": bulkResult,
			"error":      err.Error(),
		}).Error("Could not delete frequent conn entries.")
	}
}