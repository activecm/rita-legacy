package parser3

import (
	log "github.com/Sirupsen/logrus"

	"sync"

	mgo "gopkg.in/mgo.v2"

	"github.com/ocmdev/rita/parser3/fileparsetypes"
	"github.com/ocmdev/rita/parser3/parsetypes"
)

//importedData is sent to a datastore to be stored away
type importedData struct {
	broData parsetypes.BroData
	file    *fileparsetypes.IndexedFile
}

//MongoDatastore is a datastore which stores bro data in MongoDB
type MongoDatastore struct {
	dbMap      map[string]map[string]chan importedData
	bufferSize int
	session    *mgo.Session
	logger     *log.Logger
	waitgroup  *sync.WaitGroup
	mutex1     *sync.Mutex
	mutex2     *sync.Mutex
}

//NewMongoDatastore creates a datastore which stores bro data in MongoDB
func NewMongoDatastore(session *mgo.Session, bufferSize int, logger *log.Logger) *MongoDatastore {
	return &MongoDatastore{
		dbMap:      make(map[string]map[string]chan importedData),
		bufferSize: bufferSize,
		session:    session,
		logger:     logger,
		waitgroup:  new(sync.WaitGroup),
		mutex1:     new(sync.Mutex),
		mutex2:     new(sync.Mutex),
	}
}

//store a line of imported data in MongoDB
func (mongo *MongoDatastore) store(data importedData) {
	mongo.mutex1.Lock()
	collectionMap, ok := mongo.dbMap[data.file.TargetDatabase]
	if !ok {
		collectionMap = make(map[string]chan importedData)
		mongo.dbMap[data.file.TargetDatabase] = collectionMap
	}
	mongo.mutex1.Unlock()
	mongo.mutex2.Lock()
	channel, ok := collectionMap[data.file.TargetCollection]
	if !ok {
		channel = make(chan importedData)
		collectionMap[data.file.TargetCollection] = channel
		mongo.waitgroup.Add(1)
		go bulkInsertImportedData(
			channel, data.file.TargetDatabase, data.file.TargetCollection,
			data.broData.Indices(), mongo.bufferSize, mongo.session.Copy(),
			mongo.waitgroup, mongo.logger,
		)
	}
	mongo.mutex2.Unlock()
	channel <- data
}

//flush flushes the datastore
func (mongo *MongoDatastore) flush() {
	mongo.mutex1.Lock()
	mongo.mutex2.Lock()
	for _, db := range mongo.dbMap {
		for _, channel := range db {
			close(channel)
		}
	}
	mongo.mutex2.Unlock()
	mongo.mutex1.Unlock()
	mongo.waitgroup.Wait()
}

func bulkInsertImportedData(channel chan importedData, targetDB string,
	targetColl string, indices []string, bufferSize int, session *mgo.Session,
	wg *sync.WaitGroup, logger *log.Logger) {

	buffer := make([]interface{}, 0, bufferSize)
	collection := session.DB(targetDB).C(targetColl)
	for data := range channel {
		if len(buffer) == bufferSize {
			bulk := collection.Bulk()
			bulk.Unordered()
			bulk.Insert(buffer...)
			_, err := bulk.Run()
			if err != nil {
				logger.WithFields(log.Fields{
					"target_database":   targetDB,
					"target_collection": targetColl,
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
			"target_database":   targetDB,
			"target_collection": targetColl,
			"error":             err.Error(),
		}).Error("Unable to insert bulk data in MongoDB")
	}

	//ensure indices
	for _, val := range indices {
		err := collection.EnsureIndex(mgo.Index{
			Key: []string{val},
		})
		if err != nil {
			logger.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Failed to create indeces")
		}
	}
	session.Close()
	wg.Done()
}
