package parser

import "github.com/ocmdev/rita/parser/parsetypes"

//Datastore allows RITA to store bro data in a database
type Datastore interface {
	Store(*ImportedData)
	Flush()
	Index()
}

//ImportedData directs BroData to a specific database and collection
type ImportedData struct {
	BroData          parsetypes.BroData
	TargetDatabase   string
	TargetCollection string
}
