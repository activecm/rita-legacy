package freqConn

import(
	"github.com/activecm/rita/parser/parsetypes"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

func (fs *FSImporter) Insert(freqConn *freqConn, datastore) {
	datastore.Store(&ImportedData{
		BroData: &parsetypes.Freq{
			Source:          freqConn.src,
			Destination:     freqConn.dst,
			ConnectionCount: freqConn.connectionCount,
		},
		TargetDatabase:   targetDB,
		TargetCollection: resConf.T.Structure.FrequentConnTable,
	})

	if err != nil {
		logger.WithFields(log.Fields{
			"bulkResult": bulkResult,
			"error":      err.Error(),
		}).Error("Could not delete frequent conn entries.")
	}
}