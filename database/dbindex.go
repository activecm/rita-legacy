package database

import (
	"github.com/blang/semver"
	log "github.com/sirupsen/logrus"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type (
	// RITADatabaseIndex tracks registered RITA databases
	RITADatabaseIndex struct {
		dbHandle            *mgo.Session
		metaDatabaseName    string
		indexCollectionName string
		log                 *log.Logger
	}
)

// NewRITADatabaseIndex creates a collection in the MetaDatabase
// to track RITA managed databases. Most importantly, the database index
// contains information on whether databases have been imported,
// and/ or analyzed.
func NewRITADatabaseIndex(dbHandle *mgo.Session,
	metaDatabaseName string, indexCollectionName string,
	logger *log.Logger) (RITADatabaseIndex, error) {

	// ensure the index collection is indexed by name
	nameIndex := mgo.Index{
		Key:        []string{"name"},
		Unique:     true,
		DropDups:   true,
		Background: true, // This is a hold over. I don't think we want this. (LL)
		Name:       "nameindex",
	}

	//creating an index is idempotent so it won't error if the index already
	//exists. Also collections are created implicitly, so we can create
	//the collection and ensure the index exists in one fell swoop.
	err := dbHandle.DB(metaDatabaseName).C(indexCollectionName).EnsureIndex(nameIndex)
	if err != nil {
		logger.WithFields(log.Fields{
			"metadatabase":     metaDatabaseName,
			"index_collection": indexCollectionName,
			"error":            err.Error(),
		}).Error("Failed to create index for the database index")
		return RITADatabaseIndex{}, err
	}

	index := RITADatabaseIndex{
		dbHandle:            dbHandle.Copy(),
		metaDatabaseName:    metaDatabaseName,
		indexCollectionName: indexCollectionName,
		log:                 logger,
	}
	return index, nil
}

func (r *RITADatabaseIndex) openIndexCollection() *mgo.Collection {
	return r.dbHandle.Copy().DB(r.metaDatabaseName).C(r.indexCollectionName)
}

func (r *RITADatabaseIndex) closeIndexCollection(coll *mgo.Collection) {
	coll.Database.Session.Close()
}

func (r *RITADatabaseIndex) newRITADatabase(indexDoc DBMetaInfo) RITADatabase {
	return RITADatabase{
		indexDoc:            indexDoc,
		metaDatabaseName:    r.metaDatabaseName,
		indexCollectionName: r.indexCollectionName,
		log:                 r.log,
	}
}

// RegisterNewDatabase registers a new database in the RITADatabaseIndex.
// NOTE: This only creates an index record. It does not
// actually create the database
//
// You may want to call SetImportFinished/ SetAnalyzed on the returned RITADatabase.
//
// Returns a RITADatabase object for interacting with the new database,
// and an error which may occur if the database index collection
// cannot be modified.
func (r *RITADatabaseIndex) RegisterNewDatabase(name string, importVersion semver.Version) (RITADatabase, error) {
	indexColl := r.openIndexCollection()
	defer r.closeIndexCollection(indexColl)

	indexDoc := DBMetaInfo{
		Name:           name,
		ImportFinished: false,
		Analyzed:       false,
		ImportVersion:  importVersion.String(),
	}

	err := indexColl.Insert(indexDoc)

	if err != nil {
		r.log.WithFields(log.Fields{
			"error": err.Error(),
			"name":  name,
		}).Error("failed to create new db document")
		return RITADatabase{}, err
	}

	return r.newRITADatabase(indexDoc), nil
}

// GetDatabase returns a RITADatabase object for a given database name
// if the database is registered in the index. Otherwise it returns
// an error.
func (r *RITADatabaseIndex) GetDatabase(name string) (RITADatabase, error) {
	indexColl := r.openIndexCollection()
	defer r.closeIndexCollection(indexColl)
	var indexDoc DBMetaInfo
	err := indexColl.Find(bson.M{"name": name}).One(&indexDoc)
	if err != nil {
		return RITADatabase{}, err
	}
	return r.newRITADatabase(indexDoc), nil
}

// GetDatabases returns a RITADatabase object for each
// registered database or an error if the database index
// collection cannot be read.
func (r *RITADatabaseIndex) GetDatabases() ([]RITADatabase, error) {
	indexColl := r.openIndexCollection()
	defer r.closeIndexCollection(indexColl)
	var results []RITADatabase
	var currIndexDoc DBMetaInfo
	iter := indexColl.Find(nil).Iter()
	for iter.Next(&currIndexDoc) {
		results = append(results, r.newRITADatabase(currIndexDoc))
	}
	// There shouldn't be an error if there aren't any databases
	if iter.Err() == mgo.ErrNotFound {
		return results, nil
	}
	return results, iter.Err()
}

// GetUnanalyzedDatabases returns a RITADatabase object for each
// registered unanalyzed database or an error if the database index
// collection cannot be read.
func (r *RITADatabaseIndex) GetUnanalyzedDatabases() ([]RITADatabase, error) {
	indexColl := r.openIndexCollection()
	defer r.closeIndexCollection(indexColl)
	var results []RITADatabase
	var currIndexDoc DBMetaInfo
	iter := indexColl.Find(bson.M{"analyzed": false}).Iter()
	for iter.Next(&currIndexDoc) {
		results = append(results, r.newRITADatabase(currIndexDoc))
	}
	// There shouldn't be an error if all the databases are analyzed
	if iter.Err() == mgo.ErrNotFound {
		return results, nil
	}
	return results, iter.Err()
}

// GetAnalyzedDatabases returns a RITADatabase object for each
// registered analyzed database or an error if the database index
// collection cannot be read.
func (r *RITADatabaseIndex) GetAnalyzedDatabases() ([]RITADatabase, error) {
	indexColl := r.openIndexCollection()
	defer r.closeIndexCollection(indexColl)
	var results []RITADatabase
	var currIndexDoc DBMetaInfo
	iter := indexColl.Find(bson.M{"analyzed": true}).Iter()
	for iter.Next(&currIndexDoc) {
		results = append(results, r.newRITADatabase(currIndexDoc))
	}
	// There shouldn't be an error if all the databases are unanalyzed
	if iter.Err() == mgo.ErrNotFound {
		return results, nil
	}
	return results, iter.Err()
}

// Close closes the underlying connection to MongoDB
func (r *RITADatabaseIndex) Close() {
	r.dbHandle.Close()
}
