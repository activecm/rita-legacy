package database

import (
	"os"
	"sync"
	"time"

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

	// LogInfo defines information about the UpdateChecker log
	LogInfo struct {
		ID      bson.ObjectId `bson:"_id,omitempty"`   // Ident
		Time    time.Time     `bson:"LastUpdateCheck"` // Top level name of the database
		Message string        `bson:"Message"`         // Top level name of the database
		Version string        `bson:"NewestVersion"`   // Top level name of the database
	}

	// DBMetaInfo defines some information about the database
	DBMetaInfo struct {
		ID             bson.ObjectId `bson:"_id,omitempty"`   // Ident
		Name           string        `bson:"name"`            // Top level name of the database
		ImportFinished bool          `bson:"import_finished"` // Has this database been entirely imported
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

//LastCheck returns most recent version check
func (m *MetaDB) LastCheck() (time.Time, semver.Version) {
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	iter := ssn.DB(m.config.S.Bro.MetaDB).C("logs").Find(bson.M{"Message": "Checking versions..."}).Sort("-Time").Iter()

	var db LogInfo
	iter.Next(&db)

	retVersion, err := semver.ParseTolerant(db.Version)

	if err == nil {
		return db.Time, retVersion
	}

	return time.Time{}, semver.Version{}
}

// AddNewDB adds a new database to the DBMetaInfo table
func (m *MetaDB) AddNewDB(name string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	err := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).Insert(
		DBMetaInfo{
			Name:           name,
			ImportFinished: false,
			Analyzed:       false,
			ImportVersion:  m.config.S.Version,
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
	_, err := m.GetDBMetaInfo(name)
	if err != nil {
		m.log.WithFields(log.Fields{
			"database_requested": name,
			"error":              err.Error(),
		}).Error("database not found in metadata directory")
		return err
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

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

// MarkDBImported marks a database as having been completely imported
func (m *MetaDB) MarkDBImported(name string, complete bool) error {
	dbr, err := m.GetDBMetaInfo(name)

	if err != nil {
		m.log.WithFields(log.Fields{
			"database_requested": name,
			"error":              err.Error(),
		}).Error("database not found in metadata directory")
		return err
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	err = ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).
		Update(bson.M{"_id": dbr.ID}, bson.M{
			"$set": bson.D{
				{"import_finished", complete},
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

// MarkDBAnalyzed marks a database as having been analyzed
func (m *MetaDB) MarkDBAnalyzed(name string, complete bool) error {
	dbr, err := m.GetDBMetaInfo(name)

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

	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

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

//migrateDBMetaInfo is used to ensure compatibility with previous database schemas.
//migrateDBMetaInfo does not migrate any data in the database. Rather,
//it converts old DBMetaInfo representations into new ones in memory.
//We follow the reasoning that RITA should be able to read documents from
//older versions.
func migrateDBMetaInfo(inInfo DBMetaInfo) (DBMetaInfo, error) {
	inVersion, err := semver.ParseTolerant(inInfo.ImportVersion)
	if err != nil {
		return inInfo, err
	}
	if inVersion.LT(semver.Version{Major: 1, Minor: 1, Patch: 0}) {
		/*
		*	Before version 1.1.0, database records in the MetaDB lacked
		* the ImportFinished flag. The flag was introduced to prevent
		* the simultaneous import and analysis of a database.
		* See: https://github.com/activecm/rita/blob/9fd7ed84a1bad3aba879e890fad83152266c8156/database/meta.go#L26
		 */
		inInfo.ImportFinished = true
	}
	return inInfo, nil
}

// runDBMetaInfoQuery runs a MongoDB query against the MetaDB Databases Table
// and performs any necessary data migration
func (m *MetaDB) runDBMetaInfoQuery(queryDoc bson.M) ([]DBMetaInfo, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	var results []DBMetaInfo
	err := ssn.DB(m.config.S.Bro.MetaDB).C(m.config.T.Meta.DatabasesTable).Find(queryDoc).All(&results)
	if err != nil {
		return results, err
	}

	for i := range results {
		results[i], err = migrateDBMetaInfo(results[i])
		if err != nil {
			return results, err
		}
	}

	return results, nil
}

// GetDBMetaInfo returns a meta db entry. This is the only function which
// returns DBMetaInfo to code outside of meta.go.
func (m *MetaDB) GetDBMetaInfo(name string) (DBMetaInfo, error) {
	results, err := m.runDBMetaInfoQuery(bson.M{"name": name})
	if err != nil {
		return DBMetaInfo{}, err
	}
	if len(results) == 0 {
		return DBMetaInfo{}, mgo.ErrNotFound
	}
	return results[0], nil
}

// GetDatabases returns a list of databases being tracked in metadb or an empty array on failure
func (m *MetaDB) GetDatabases() []string {
	dbs, err := m.runDBMetaInfoQuery(nil)
	if err != nil {
		return nil
	}
	var results []string
	for _, db := range dbs {
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
	dbs, err := m.runDBMetaInfoQuery(bson.M{"analyzed": false})
	if err != nil {
		return nil
	}
	var results []string
	for _, db := range dbs {
		results = append(results, db.Name)
	}
	return results
}

// GetAnalyzeReadyDatabases builds a list of database names which are ready to be analyzed
func (m *MetaDB) GetAnalyzeReadyDatabases() []string {
	//note import_finished is queried as {"$ne": false} rather than just true
	//since prior to version 1.1.0, the field did not exist.
	dbs, err := m.runDBMetaInfoQuery(
		bson.M{
			"analyzed":        false,
			"import_finished": bson.M{"$ne": false},
		},
	)
	if err != nil {
		return nil
	}
	var results []string
	for _, db := range dbs {
		results = append(results, db.Name)
	}
	return results
}

// GetAnalyzedDatabases builds a list of database names which have been analyzed
func (m *MetaDB) GetAnalyzedDatabases() []string {
	dbs, err := m.runDBMetaInfoQuery(bson.M{"analyzed": true})
	if err != nil {
		return nil
	}
	var results []string
	for _, db := range dbs {
		results = append(results, db.Name)
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
