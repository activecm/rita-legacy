package uconn

import(
	"github.com/activecm/rita/parser/parsetypes"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

func (fs *FSImporter) Insert(uconn *Uconn) error {
	resDB := fs.res.DB
	resConf := fs.res.Config
	logger := fs.res.Log

	ssn := resDB.Session.Copy()
	defer ssn.Close()

	fmt.Println("\t[-] Creating Uconns and Hosts Collections. This may take a while.")
	// add uconn pair to uconn table
	datastore.Store(&ImportedData{
		BroData: &parsetypes.Uconn{
			Source:           uconn.src,
			Destination:      uconn.dst,
			ConnectionCount:  uconn.connectionCount,
			LocalSource:      uconn.isLocalSrc,
			LocalDestination: uconn.isLocalDst,
			TotalBytes:       uconn.totalBytes,
			AverageBytes:     uconn.avgBytes,
			MaxDuration:      uconn.maxDuration,
			TSList:           uconn.tsList,
			OrigBytesList:    uconn.origBytesList,
		},
		TargetDatabase:   targetDB,
		TargetCollection: resConf.T.Structure.UniqueConnTable,
	})
}