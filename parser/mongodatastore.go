package parser

import (
	"errors"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/activecm/rita/database"
	mgo "github.com/globalsign/mgo"
)

//storeMap maps database names to collection maps and provides a mutex
//to sync around
type storeMap struct {
	databases map[string]*collectionMap
	rwLock    *sync.Mutex
}

//collectionMap maps collection names to collection writers and provides a
//mutex to sync around
type collectionMap struct {
	collections map[string]*collectionWriter
	rwLock      *sync.Mutex
}

//collectionWriter reads a channel and inserts the data into MongoDB
type collectionWriter struct {
	writeChannel     chan *ImportedData
	writerWG         *sync.WaitGroup
	session          *mgo.Session
	logger           *log.Logger
	bufferSize       int
	targetDatabase   string
	targetCollection string
	indices          []string
}

//MongoDatastore provides a backend for storing bro data in MongoDB
type MongoDatastore struct {
	session       *mgo.Session
	metaDB        *database.MetaDB
	bufferSize    int
	logger        *log.Logger
	writerWG      *sync.WaitGroup
	writeMap      storeMap
	analyzedDBs   []string
	unanalyzedDBs []string
}

//NewMongoDatastore returns a new MongoDatastore and caches the existing
//db names
func NewMongoDatastore(session *mgo.Session, metaDB *database.MetaDB,
	bufferSize int, logger *log.Logger) *MongoDatastore {
	return &MongoDatastore{
		session:    session,
		metaDB:     metaDB,
		bufferSize: bufferSize,
		logger:     logger,
		writerWG:   new(sync.WaitGroup),
		writeMap: storeMap{
			databases: make(map[string]*collectionMap),
			rwLock:    new(sync.Mutex),
		},
		analyzedDBs:   metaDB.GetAnalyzedDatabases(),
		unanalyzedDBs: metaDB.GetUnAnalyzedDatabases(),
	}
}

//Store saves parsed Bro data to MongoDB.
//Additionally, it caches some information to create indices later on
func (mongo *MongoDatastore) Store(data *ImportedData) {
	collMap, err := mongo.getCollectionMap(data)
	if err != nil {
		mongo.logger.Error(err)
		return
	}
	collWriter := mongo.getCollectionWriter(data, collMap)
	collWriter.writeChannel <- data
}

//Flush waits for all writing to finish
func (mongo *MongoDatastore) Flush() {
	mongo.writeMap.rwLock.Lock()
	for _, collMap := range mongo.writeMap.databases {
		collMap.rwLock.Lock()
		for _, collWriter := range collMap.collections {
			close(collWriter.writeChannel)
		}
		collMap.rwLock.Unlock()
	}
	mongo.writeMap.rwLock.Unlock()
	mongo.writerWG.Wait()
}

//Index ensures that the data is searchable
func (mongo *MongoDatastore) Index() {
	//NOTE: We do this one by one in order to prevent individual indexing
	//operations from taking too long
	ssn := mongo.session.Copy()
	defer ssn.Close()

	mongo.writeMap.rwLock.Lock()
	for _, collMap := range mongo.writeMap.databases {
		collMap.rwLock.Lock()
		for _, collWriter := range collMap.collections {
			collection := ssn.DB(collWriter.targetDatabase).C(collWriter.targetCollection)
			for _, index := range collWriter.indices {
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
		collMap.rwLock.Unlock()
	}
	mongo.writeMap.rwLock.Unlock()
}

//getCollectionMap returns a map from collection names to collection writers
//given a bro entry's target database. If the database does not exist,
//getCollectionMap will create the database. If the database does exist
//and the database has been analyzed, getCollectionMap will return an error.
func (mongo *MongoDatastore) getCollectionMap(data *ImportedData) (*collectionMap, error) {
	mongo.writeMap.rwLock.Lock()
	defer mongo.writeMap.rwLock.Unlock()

	//check the cache for the collection map
	collMap, ok := mongo.writeMap.databases[data.TargetDatabase]
	if ok {
		return collMap, nil
	}

	//check if the database is already analyzed

	//iterate over indices to save RAM
	//nolint: golint
	for i, _ := range mongo.analyzedDBs {
		if mongo.analyzedDBs[i] == data.TargetDatabase {
			return nil, errors.New("cannot import bro data into already analyzed database")
		}
	}

	//check if the database was created in an earlier parse
	targetDBExists := false
	//nolint: golint
	for i, _ := range mongo.unanalyzedDBs {
		if mongo.unanalyzedDBs[i] == data.TargetDatabase {
			targetDBExists = true
		}
	}

	if targetDBExists {
		compatible, err := mongo.metaDB.CheckCompatibleImport(data.TargetDatabase)
		if err != nil {
			return nil, err
		}
		if !compatible {
			return nil, errors.New("cannot import bro data into already populated, incompatible database")
		}
	} else {
		//create the database if it doesn't exist
		err := mongo.metaDB.AddNewDB(data.TargetDatabase)
		if err != nil {
			return nil, err
		}
		mongo.unanalyzedDBs = append(mongo.unanalyzedDBs, data.TargetDatabase)
	}

	mongo.writeMap.databases[data.TargetDatabase] = &collectionMap{
		collections: make(map[string]*collectionWriter),
		rwLock:      new(sync.Mutex),
	}
	return mongo.writeMap.databases[data.TargetDatabase], nil
}

//getCollectionWriter returns a collection writer which can be used to send
//data to a specific MongoDB collection. If a collection writer does not exist
//in the cache, it is created and a new thread is spun up for it.
func (mongo *MongoDatastore) getCollectionWriter(data *ImportedData, collMap *collectionMap) *collectionWriter {
	collMap.rwLock.Lock()
	defer collMap.rwLock.Unlock()
	collWriter, ok := collMap.collections[data.TargetCollection]
	if ok {
		return collWriter
	}
	collMap.collections[data.TargetCollection] = &collectionWriter{
		writeChannel:     make(chan *ImportedData),
		writerWG:         mongo.writerWG,
		session:          mongo.session.Copy(),
		logger:           mongo.logger,
		bufferSize:       mongo.bufferSize,
		targetDatabase:   data.TargetDatabase,
		targetCollection: data.TargetCollection,
		indices:          data.BroData.Indices(),
	}
	go collMap.collections[data.TargetCollection].bulkInsert()
	return collMap.collections[data.TargetCollection]
}

//bulkInsert is a goroutine which reads a channel and inserts the data in bulk
//into MongoDB
func (writer *collectionWriter) bulkInsert() {
	writer.writerWG.Add(1)
	defer writer.writerWG.Done()
	defer writer.session.Close()

	buffer := make([]interface{}, 0, writer.bufferSize)
	collection := writer.session.DB(writer.targetDatabase).C(writer.targetCollection)

	for data := range writer.writeChannel {
		if len(buffer) == writer.bufferSize {
			bulk := collection.Bulk()
			bulk.Unordered()
			bulk.Insert(buffer...)
			_, err := bulk.Run()
			if err != nil {
				writer.logger.WithFields(log.Fields{
					"target_database":   writer.targetDatabase,
					"target_collection": writer.targetCollection,
					"error":             err.Error(),
				}).Error("Unable to insert bulk data in MongoDB")
			}
			buffer = buffer[:0]
		}
		buffer = append(buffer, data.BroData)
	}

	//guaranteed to be at least 1 line in the buffer
	bulk := collection.Bulk()
	bulk.Unordered()
	bulk.Insert(buffer...)
	_, err := bulk.Run()
	if err != nil {
		writer.logger.WithFields(log.Fields{
			"target_database":   writer.targetDatabase,
			"target_collection": writer.targetCollection,
			"error":             err.Error(),
		}).Error("Unable to insert bulk data in MongoDB")
	}
}
