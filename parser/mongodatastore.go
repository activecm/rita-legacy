package parser

import (
	log "github.com/sirupsen/logrus"

	"sync"

	mgo "gopkg.in/mgo.v2"

	"github.com/ocmdev/rita/database"
	fpt "github.com/ocmdev/rita/parser/fileparsetypes"
	pt "github.com/ocmdev/rita/parser/parsetypes"
)

//importedData is sent to a datastore to be stored away
type importedData struct {
	broData        pt.BroData
	targetDatabase string
	file           *fpt.IndexedFile
}

//collectionStore binds a collection write channel with the target collection
//and the indices to be applied to it
type collectionStore struct {
	writeChannel chan importedData
	database     string
	collection   string
	indices      []string
}

//MongoDatastore is a datastore which stores bro data in MongoDB
type MongoDatastore struct {
	dbMap       map[string]map[string]collectionStore
	existingDBs []string
	metaDB      *database.MetaDBHandle
	bufferSize  int
	session     *mgo.Session
	logger      *log.Logger
	waitgroup   *sync.WaitGroup
	mutex1      *sync.Mutex
	mutex2      *sync.Mutex
}

//NewMongoDatastore creates a datastore which stores bro data in MongoDB
func NewMongoDatastore(session *mgo.Session, metaDB *database.MetaDBHandle,
	bufferSize int, logger *log.Logger) *MongoDatastore {
	return &MongoDatastore{
		dbMap:       make(map[string]map[string]collectionStore),
		existingDBs: metaDB.GetDatabases(),
		metaDB:      metaDB,
		bufferSize:  bufferSize,
		session:     session,
		logger:      logger,
		waitgroup:   new(sync.WaitGroup),
		mutex1:      new(sync.Mutex), //mutex1 syncs the first level of map access
		mutex2:      new(sync.Mutex), //mutex2 syncs the second level of map access
		//NOTE: Mutex2 may be replaced with a map of mutexes for better performance
	}
}

//store a line of imported data in MongoDB
func (mongo *MongoDatastore) store(data importedData) {
	//get the map representing the target database
	mongo.mutex1.Lock()
	collectionMap, ok := mongo.dbMap[data.targetDatabase]
	if !ok {
		mongo.registerDatabase(data.targetDatabase)
		collectionMap = make(map[string]collectionStore)
		mongo.dbMap[data.targetDatabase] = collectionMap
	}
	mongo.mutex1.Unlock()

	//get the collectionStore for the target collection
	mongo.mutex2.Lock()
	coll, ok := collectionMap[data.file.TargetCollection]
	if !ok {
		coll = collectionStore{
			writeChannel: make(chan importedData),
			database:     data.targetDatabase,
			collection:   data.file.TargetCollection,
			indices:      data.broData.Indices(),
		}
		collectionMap[data.file.TargetCollection] = coll
		//start the goroutine for this writer
		mongo.waitgroup.Add(1)
		go bulkInsertImportedData(
			coll, mongo.bufferSize, mongo.session.Copy(),
			mongo.waitgroup, mongo.logger,
		)
	}
	mongo.mutex2.Unlock()
	//queue up the line to be written
	coll.writeChannel <- data
}

//flush flushes the datastore
func (mongo *MongoDatastore) flush() {
	//wait for any changes to the collection maps to finish
	mongo.mutex1.Lock()
	mongo.mutex2.Lock()
	//close out the write channels, allowing them to flush
	for _, db := range mongo.dbMap {
		for _, collStore := range db {
			close(collStore.writeChannel)
		}
	}
	mongo.mutex2.Unlock()
	mongo.mutex1.Unlock()
	//wait for the channels to flush
	mongo.waitgroup.Wait()
}

//finalize ensures the indexes are applied to the mongo collections
func (mongo *MongoDatastore) finalize() {
	//ensure indices
	//NOTE: We do this one by one in order to prevent individual indexing
	//operations from taking too long
	ssn := mongo.session.Copy()
	defer ssn.Close()
	//wait for any changes to the collection maps to finish
	//this shouldn't be an issue but it doesn't hurt
	mongo.mutex1.Lock()
	mongo.mutex2.Lock()
	for _, collMap := range mongo.dbMap {
		for _, collStore := range collMap {
			collection := ssn.DB(collStore.database).C(collStore.collection)
			for _, index := range collStore.indices {
				err := collection.EnsureIndex(mgo.Index{
					Key: []string{index},
				})
				if err != nil {
					mongo.logger.WithFields(log.Fields{
						"error": err.Error(),
					}).Error("Failed to create indeces")
				}
			}
		}
	}
	mongo.mutex2.Unlock()
	mongo.mutex1.Unlock()
}

func (mongo *MongoDatastore) registerDatabase(db string) {
	found := false
	for _, existingDB := range mongo.existingDBs {
		if db == existingDB {
			found = true
			break
		}
	}
	if !found {
		mongo.metaDB.AddNewDB(db)
	} else {
		mongo.logger.Error("Attempted to insert data into existing database.")
		panic("[!] Attempted to insert data into existing database.")
	}
}

func bulkInsertImportedData(coll collectionStore, bufferSize int,
	session *mgo.Session, wg *sync.WaitGroup, logger *log.Logger) {

	//buffer the writes to MongoDB
	buffer := make([]interface{}, 0, bufferSize)
	collection := session.DB(coll.database).C(coll.collection)

	//append data to the buffer until it is full, then insert them
	for data := range coll.writeChannel {
		if len(buffer) == bufferSize {
			bulk := collection.Bulk()
			bulk.Unordered()
			bulk.Insert(buffer...)
			_, err := bulk.Run()
			if err != nil {
				logger.WithFields(log.Fields{
					"target_database":   coll.database,
					"target_collection": coll.collection,
					"error":             err.Error(),
				}).Error("Unable to insert bulk data in MongoDB")
			}
			buffer = buffer[:0]
		}
		buffer = append(buffer, data.broData)
	}

	//guaranteed to be at least 1 line in the buffer
	bulk := collection.Bulk()
	bulk.Unordered()
	bulk.Insert(buffer...)
	_, err := bulk.Run()
	if err != nil {
		logger.WithFields(log.Fields{
			"target_database":   coll.database,
			"target_collection": coll.collection,
			"error":             err.Error(),
		}).Error("Unable to insert bulk data in MongoDB")
	}
	session.Close()
	wg.Done()
}
