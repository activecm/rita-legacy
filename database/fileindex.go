package database

import (
	"github.com/activecm/rita/parser/fileparsetypes"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	//ImportedFilesIndex maintains a persistent index over
	//the files that RITA has imported. This aids in preventing
	//duplicate imports.
	ImportedFilesIndex struct {
		dbHandle            *mgo.Session
		metaDatabaseName    string
		indexCollectionName string
		log                 *log.Logger
	}
)

//NewImportedFilesIndex creates a new ImportedFilesIndex
//and sets up a backing MongoDB collection to maintain the index in.
//The backing collection will be stored in the MetaDatabase
//given by metaDatabaseName. The collection will be
//called indexCollectionName.
func NewImportedFilesIndex(dbHandle *mgo.Session,
	metaDatabaseName string, indexCollectionName string,
	logger *log.Logger) (ImportedFilesIndex, error) {

	//ensure the index collection is indexed by file hash + database
	fileIndex := mgo.Index{
		Key:        []string{"hash", "database"},
		Unique:     true,
		DropDups:   true,
		Background: true, //this is a hold over. I don't think we want this. (LL)
		Name:       "hashindex",
	}

	//creating an index is idempotent so it won't error if the index already
	//exists. Also collections are created implicitly, so we can create
	//the collection and ensure the index exists in one fell swoop.
	err := dbHandle.DB(metaDatabaseName).C(indexCollectionName).EnsureIndex(fileIndex)
	if err != nil {
		logger.WithFields(log.Fields{
			"metadatabase":     metaDatabaseName,
			"index_collection": indexCollectionName,
			"error":            err.Error(),
		}).Error("Failed to create index for the imported file index")
		return ImportedFilesIndex{}, err
	}

	index := ImportedFilesIndex{
		dbHandle:            dbHandle.Copy(),
		metaDatabaseName:    metaDatabaseName,
		indexCollectionName: indexCollectionName,
		log:                 logger,
	}

	return index, nil
}

func (i *ImportedFilesIndex) openIndexCollection() *mgo.Collection {
	return i.dbHandle.Copy().DB(i.metaDatabaseName).C(i.indexCollectionName)
}

func (i *ImportedFilesIndex) closeIndexCollection(coll *mgo.Collection) {
	coll.Database.Session.Close()
}

// GetFiles retrieves a list containing all of the files registered in the
// ImportedFilesIndex. In the case of failure, the function returns a nil
// array and an error.
func (i *ImportedFilesIndex) GetFiles() ([]fileparsetypes.IndexedFile, error) {
	indexColl := i.openIndexCollection()
	defer i.closeIndexCollection(indexColl)

	var indexedFiles []fileparsetypes.IndexedFile
	err := indexColl.Find(nil).Iter().All(&indexedFiles)
	if err != nil {
		i.log.WithFields(log.Fields{
			"metadatabase":     i.metaDatabaseName,
			"index_collection": i.indexCollectionName,
			"error":            err.Error(),
		}).Error("could not retrieve files from file index")
		return nil, err
	}
	return indexedFiles, nil
}

// RegisterFiles registers files in the ImportedFilesIndex using the MongoDB bulk API
func (i *ImportedFilesIndex) RegisterFiles(files []*fileparsetypes.IndexedFile) error {
	indexColl := i.openIndexCollection()
	defer i.closeIndexCollection(indexColl)

	bulk := indexColl.Bulk()
	bulk.Unordered()

	//construct the interface slice for bulk
	interfaceSlice := make([]interface{}, len(files))
	for i, file := range files {
		interfaceSlice[i] = *file
	}

	bulk.Insert(interfaceSlice...)
	_, err := bulk.Run()
	if err != nil {
		i.log.WithFields(log.Fields{
			"metadatabase":     i.metaDatabaseName,
			"index_collection": i.indexCollectionName,
			"error":            err.Error(),
		}).Error("could not register files in imported files index")
		return err
	}
	return nil
}

//RemoveFilesForDatabase removes all files in the ImportedFilesIndex
//which were imported into a specified database
func (i *ImportedFilesIndex) RemoveFilesForDatabase(database string) error {
	_, err := i.dbHandle.DB(i.metaDatabaseName).C(i.indexCollectionName).RemoveAll(
		bson.M{"database": database},
	)
	//TODO: verify this doesn't return an error if the query doesn't match any records
	return err
}

// Close closes the underlying connection to MongoDB
func (i *ImportedFilesIndex) Close() {
	i.dbHandle.Close()
}
