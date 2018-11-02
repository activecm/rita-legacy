package database

import (
	"os"
	"sync"

	"github.com/activecm/rita/config"
	fpt "github.com/activecm/rita/parser/fileparsetypes"
	"github.com/blang/semver"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (

	// MetaDB exports control for the meta database
	MetaDB struct {
		lock     *sync.Mutex    // Read and write lock
		config   *config.Config // configuration info
		dbHandle *mgo.Session   // Database handle
		log      *log.Logger    // Logging object
	}

	// DBMetaInfo defines some information about the database
	DBMetaInfo struct {
		ID             bson.ObjectId `bson:"_id,omitempty"`   // Ident
		Name           string        `bson:"name"`            // Top level name of the database
		Analyzed       bool          `bson:"analyzed"`        // Has this database been analyzed
		ImportVersion  string        `bson:"import_version"`  // Rita version at import
		AnalyzeVersion string        `bson:"analyze_version"` // Rita version at analyze
	}
)

// NewMetaDB instantiates a new handle for the RITA MetaDatabase
func NewMetaDB(config *config.Config, dbHandle *mgo.Session,
	log *log.Logger) *MetaDB {
	metaDB := &MetaDB{
		lock:     new(sync.Mutex),
		config:   config,
		dbHandle: dbHandle,
		log:      log,
	}
	//Build Meta collection
	if !metaDB.isBuilt() {
		metaDB.createMetaDB()
	}
	return metaDB
}

// AddNewDB adds a new database to the DBMetaInfo table
func (m *MetaDB) AddNewDB(name string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	err := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).Insert(
		DBMetaInfo{
			Name:          name,
			Analyzed:      false,
			ImportVersion: m.config.S.Version,
		},
	)
	if err != nil {
		m.log.WithFields(log.Fields{
			"error": err.Error(),
			"name":  name,
		}).Error("failed to create new db document")
		return err
	}

	return nil
}

// DeleteDB removes a database managed by RITA
func (m *MetaDB) DeleteDB(name string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	//get the record
	var db DBMetaInfo
	err := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).Find(bson.M{"name": name}).One(&db)
	if err != nil {
		return err
	}

	//delete the record
	err = ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).Remove(bson.M{"name": name})
	if err != nil {
		return err
	}

	//drop the data
	ssn.DB(name).DropDatabase()

	//delete any parsed file records associated
	_, err = ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.FilesTable).RemoveAll(bson.M{"database": name})
	if err != nil {
		return err
	}

	_, err = ssn.DB(m.config.S.Bro.MetaDB).C("files").RemoveAll(bson.M{"database": name})
	if err != nil {
		return err
	}

	return nil
}

// MarkDBAnalyzed marks a database as having been analyzed
func (m *MetaDB) MarkDBAnalyzed(name string, complete bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	dbr := DBMetaInfo{}
	err := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).
		Find(bson.M{"name": name}).One(&dbr)

	if err != nil {
		m.log.WithFields(log.Fields{
			"database_requested": name,
			"error":              err.Error(),
		}).Error("database not found in metadata directory")
		return err
	}

	var versionTag string
	if complete {
		versionTag = m.config.S.Version
	} else {
		versionTag = ""
	}

	err = ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).
		Update(bson.M{"_id": dbr.ID}, bson.M{
			"$set": bson.D{
				{"analyzed", complete},
				{"analyze_version", versionTag},
			},
		})

	if err != nil {
		m.log.WithFields(log.Fields{
			"metadb_attempted":   m.config.S.Bro.MetaDB,
			"database_requested": name,
			"_id":                dbr.ID.Hex,
			"error":              err.Error(),
		}).Error("could not update database entry in meta")
		return err
	}
	return nil
}

// GetDBMetaInfo returns a meta db entry
func (m *MetaDB) GetDBMetaInfo(name string) (DBMetaInfo, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.dbHandle.Copy()
	defer ssn.Close()
	var result DBMetaInfo
	err := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).Find(bson.M{"name": name}).One(&result)
	return result, err
}

// GetDatabases returns a list of databases being tracked in metadb or an empty array on failure
func (m *MetaDB) GetDatabases() []string {
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	iter := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).Find(nil).Iter()

	var results []string
	var db DBMetaInfo
	for iter.Next(&db) {
		results = append(results, db.Name)
	}
	return results
}

//CheckCompatibleImport checks if a database was imported with a version of
//RITA which is compatible with the running version
func (m *MetaDB) CheckCompatibleImport(targetDatabase string) (bool, error) {
	dbData, err := m.GetDBMetaInfo(targetDatabase)
	if err != nil {
		return false, err
	}
	existingVer, err := semver.ParseTolerant(dbData.ImportVersion)
	if err != nil {
		return false, err
	}
	return m.config.R.Version.Major == existingVer.Major, nil
}

//CheckCompatibleAnalyze checks if a database was analyzed with a version of
//RITA which is compatible with the running version
func (m *MetaDB) CheckCompatibleAnalyze(targetDatabase string) (bool, error) {
	dbData, err := m.GetDBMetaInfo(targetDatabase)
	if err != nil {
		return false, err
	}
	existingVer, err := semver.ParseTolerant(dbData.AnalyzeVersion)
	if err != nil {
		return false, err
	}
	return m.config.R.Version.Major == existingVer.Major, nil
}

// GetUnAnalyzedDatabases builds a list of database names which have yet to be analyzed
func (m *MetaDB) GetUnAnalyzedDatabases() []string {
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	var results []string
	var cur DBMetaInfo
	iter := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).Find(bson.M{"analyzed": false}).Iter()
	for iter.Next(&cur) {
		results = append(results, cur.Name)
	}
	return results
}

// GetAnalyzedDatabases builds a list of database names which have been analyzed
func (m *MetaDB) GetAnalyzedDatabases() []string {
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	var results []string
	var cur DBMetaInfo
	iter := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).Find(bson.M{"analyzed": true}).Iter()
	for iter.Next(&cur) {
		results = append(results, cur.Name)
	}
	return results
}

///////////////////////////////////////////////////////////////////////////////
//                            File Processing                                //
///////////////////////////////////////////////////////////////////////////////

// GetFiles gets a list of all IndexedFile objects in the database if successful return a list of files
// from the database, in the case of failure return a zero length list of files and generat a log
// message.
func (m *MetaDB) GetFiles() ([]fpt.IndexedFile, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	var toReturn []fpt.IndexedFile

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	err := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.FilesTable).
		Find(nil).Iter().All(&toReturn)
	if err != nil {
		m.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("could not fetch files from meta database")
		return nil, err
	}
	return toReturn, nil
}

//AddParsedFiles adds indexed files to the files the metaDB using the bulk API
func (m *MetaDB) AddParsedFiles(files []*fpt.IndexedFile) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if len(files) == 0 {
		return nil
	}
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	bulk := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.FilesTable).Bulk()
	bulk.Unordered()

	//construct the interface slice for bulk
	interfaceSlice := make([]interface{}, len(files))
	for i, d := range files {
		interfaceSlice[i] = *d
	}

	bulk.Insert(interfaceSlice...)
	_, err := bulk.Run()
	if err != nil {
		m.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("could not insert files into meta database")
		return err
	}
	return nil
}

/////////////////////

// isBuilt checks to see if a file table exists, as the existence of parsed files is prerequisite
// to the existence of anything else.
func (m *MetaDB) isBuilt() bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	coll, err := ssn.DB(m.config.S.Bro.MetaDB).CollectionNames()
	if err != nil {
		m.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("error when looking up metadata collections")
		return false
	}

	for _, name := range coll {
		if name == m.config.T.Meta.FilesTable {
			return true
		}
	}

	return false
}

// createMetaDB creates a new metadata database failure is not an option,
// if this function fails it will bring down the system.
func (m *MetaDB) createMetaDB() {
	m.lock.Lock()
	defer m.lock.Unlock()

	errchk := func(err error) {
		if err == nil {
			return
		}
		m.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("newMetaDBHandle failed to build database (aborting)")
		os.Exit(-1)
	}

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	// Create the files collection
	myCol := mgo.CollectionInfo{
		DisableIdIndex: false,
		Capped:         false,
	}

	err := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Log.RitaLogTable).Create(&myCol)
	errchk(err)

	err = ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.FilesTable).Create(&myCol)
	errchk(err)

	idx := mgo.Index{
		Key:        []string{"hash", "database"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Name:       "hashindex",
	}

	err = ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.FilesTable).EnsureIndex(idx)
	errchk(err)

	// Create the database collection
	err = ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).Create(&myCol)
	errchk(err)

	idx = mgo.Index{
		Key:        []string{"name"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Name:       "nameindex",
	}

	err = ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).EnsureIndex(idx)
	errchk(err)

}
