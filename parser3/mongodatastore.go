package parser3

import "github.com/ocmdev/rita/parser3/parsetypes"

//MongoDatastore is a datastore which stores bro data in MongoDB
type MongoDatastore struct {
	dbMap map[string]map[string]chan ImportedData
}

//ImportedData is sent to a datastore to be stored away
type ImportedData struct {
	BroData parsetypes.BroData
	File    *IndexedFile
}

//Datastore represents a place to store imported bro data
type Datastore interface {
	Store(ImportedData)
}

//NewMongoDatastore creates a datastore which stores bro data in MongoDB
func NewMongoDatastore() *MongoDatastore {
	return &MongoDatastore{
		dbMap: make(map[string]map[string]chan ImportedData),
	}
}

//Store a line of imported data in MongoDB
func (mongo *MongoDatastore) Store(data ImportedData) {
	collectionMap, ok := mongo.dbMap[data.File.TargetDatabase]
	if !ok {
		collectionMap = make(map[string]chan ImportedData)
		mongo.dbMap[data.File.TargetDatabase] = collectionMap
	}
	channel, ok := collectionMap[data.File.TargetCollection]
	if !ok {
		channel = make(chan ImportedData)
		collectionMap[data.File.TargetCollection] = channel
		go mongo.bulkInsertImportedData(channel)
	}
	channel <- data
}

func (mongo *MongoDatastore) bulkInsertImportedData(channel chan ImportedData) {
	for data := range channel {
		//TODO: Write mongo bulk insertion code
		_ = data
	}
}
