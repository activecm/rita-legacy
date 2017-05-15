package parser3

import (
	"sync"

	"github.com/ocmdev/rita/parser3/parsetypes"
)

//importedData is sent to a datastore to be stored away
type importedData struct {
	broData parsetypes.BroData
	file    *IndexedFile
}

//MongoDatastore is a datastore which stores bro data in MongoDB
type MongoDatastore struct {
	dbMap     map[string]map[string]chan importedData
	waitgroup *sync.WaitGroup
}

//NewMongoDatastore creates a datastore which stores bro data in MongoDB
func NewMongoDatastore() *MongoDatastore {
	return &MongoDatastore{
		dbMap:     make(map[string]map[string]chan importedData),
		waitgroup: new(sync.WaitGroup),
	}
}

//store a line of imported data in MongoDB
func (mongo *MongoDatastore) store(data importedData) {
	collectionMap, ok := mongo.dbMap[data.file.TargetDatabase]
	if !ok {
		collectionMap = make(map[string]chan importedData)
		mongo.dbMap[data.file.TargetDatabase] = collectionMap
	}
	channel, ok := collectionMap[data.file.TargetCollection]
	if !ok {
		channel = make(chan importedData)
		collectionMap[data.file.TargetCollection] = channel
		mongo.waitgroup.Add(1)
		go bulkInsertImportedData(channel, mongo.waitgroup)
	}
	channel <- data
}

//flush flushes the datastore
func (mongo *MongoDatastore) flush() {
	for _, db := range mongo.dbMap {
		for _, channel := range db {
			close(channel)
		}
	}
	mongo.waitgroup.Wait()
}

func bulkInsertImportedData(channel chan importedData, wg *sync.WaitGroup) {
	for data := range channel {
		_ = data
		//TODO: Write mongo bulk insertion code
		//jsonData, _ := json.Marshal(data.broData)
		//jsonData2, _ := json.Marshal(data.file)
		//fmt.Println(string(jsonData))
		//fmt.Println(string(jsonData2))
		//fmt.Println("")
	}
	wg.Done()
}
