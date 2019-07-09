package database

import (
	"strconv"
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

	// Range defines a min and max value
	Range struct {
		Min int64 `bson:"min"`
		Max int64 `bson:"max"`
	}

	// DBMetaInfo defines some information about the database
	DBMetaInfo struct {
		ID             bson.ObjectId `bson:"_id,omitempty"`   // Ident
		Name           string        `bson:"name"`            // Top level name of the database
		Analyzed       bool          `bson:"analyzed"`        // Has this database been analyzed
		AnalyzeVersion string        `bson:"analyze_version"` // Rita version at analyze
		Rolling        bool          `bson:"rolling"`
		TotalChunks    int           `bson:"total_chunks"`
		CurrentChunk   int           `bson:"current_chunk"`
		TsRange        Range         `bson:"ts_range"`
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

	return metaDB
}

//GetRollingSettings gets the current rolling settings
func (m *MetaDB) GetRollingSettings(db string) (exists bool, isRolling bool, currChunk int, totalChunks int, err error) {
	// pull down dataset record from metadatabase
	result, err := m.GetDBMetaInfo(db)
	if err != nil && err != mgo.ErrNotFound {
		return
	}

	if err != mgo.ErrNotFound {
		exists = true
	}
	err = nil
	isRolling = result.Rolling
	currChunk = result.CurrentChunk
	totalChunks = result.TotalChunks
	return
}

//SetRollingSettings ensures that a given db is marked as rolling,
//ensures that total_chunks matches numchunks, and sets the current_chunk to chunk.
func (m *MetaDB) SetRollingSettings(db string, chunk int, numchunks int) error {
	// pull down dataset record from metadatabase
	result, err := m.GetDBMetaInfo(db)
	if err != nil {
		return err
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	if !result.Rolling {
		m.log.Infof("The dataset [ %s ] is being converted to a rolling dataset.", db)
	} else if result.TotalChunks != numchunks {
		m.log.Warnf("The total chunk size for existing rolling dataset [ %s ] was set to [ %d ] and is being changed to [ %d ].",
			db, result.TotalChunks, numchunks)
	}

	// update rolling settings
	_, err = ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).
		Upsert(
			// selector
			bson.M{"name": db},
			// data
			bson.M{"$set": bson.M{
				"rolling": true,
				"current_chunk": chunk,
				"total_chunks": numchunks},
			},
		)
	if err != nil {
		return err
	}

	return nil
}

//LastCheck returns most recent version check
func (m *MetaDB) LastCheck() (time.Time, semver.Version) {
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	iter := ssn.DB(m.config.S.MongoDB.MetaDB).C("logs").Find(bson.M{"Message": "Checking versions..."}).Sort("-Time").Iter()

	var db LogInfo
	iter.Next(&db)

	retVersion, err := semver.ParseTolerant(db.Version)

	if err == nil {
		return db.Time, retVersion
	}

	return time.Time{}, semver.Version{}
}

// AddNewDB adds a new database to the DBMetaInfo table. All new databases are
// ready to be rolling databases.
func (m *MetaDB) AddNewDB(name string, currentChunk, totalChunks int) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	err := ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).Insert(
		DBMetaInfo{
			Name:           name,
			Analyzed:       false,
			AnalyzeVersion: m.config.S.Version,
			Rolling:        false,
			CurrentChunk:   currentChunk,
			TotalChunks:    totalChunks,
		},
	)
	if err != nil {
		m.log.WithFields(log.Fields{
			"error": err.Error(),
			"name":  name,
		}).Error("failed to create new db document")
		return err
	}

	// Used to initialize Mongo array
	// e.g. [{"set": false}, {"set": false}]
    cidList := make([]struct {Set bool `bson:"set"`}, totalChunks)

	_, err = ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).
    	Upsert(
    		// selector
			bson.M{"name": name},
			// data
			bson.M{"$set": bson.M{
				"cid_list": cidList,
			}},
		)
	if err != nil {
		return err
	}

	return nil
}

//DBExists returns whether or not a metadatabase record has been created for a database
func (m *MetaDB) DBExists(name string) (bool, error) {
	_, err := m.GetDBMetaInfo(name)
	if err != nil && err != mgo.ErrNotFound {
		return false, err
	}
	if err == mgo.ErrNotFound {
		return false, nil
	}
	return true, nil
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
	_, err = ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).RemoveAll(bson.M{"name": name})
	if err != nil {
		return err
	}

	// TODO: this is left in for backwards compatibility
	//delete any parsed file records associated
	_, err = ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.FilesTable).RemoveAll(bson.M{"database": name})
	if err != nil {
		return err
	}

	return nil
}

// GetTSRange adds the min and max timestamps for current dataset
func (m *MetaDB) GetTSRange(name string) (int64, int64, error) {
	dbr, err := m.GetDBMetaInfo(name)

	var min, max int64

	if err != nil {
		m.log.WithFields(log.Fields{
			"database_requested": name,
			"error":              err.Error(),
		}).Error("Could not add timestamp range: database not found in metadata directory")
		return min, max, err
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	type tsInfo struct {
		Min int64 `bson:"min" json:"min"`
		Max int64 `bson:"max" json:"max"`
	}

	var tsRes struct {
		TSRange tsInfo `bson:"ts_range"`
	}

	// get min and max timestamps
	err = ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).Find(bson.M{"name": name}).One(&tsRes)

	if err != nil {
		m.log.WithFields(log.Fields{
			"metadb_attempted":   m.config.S.MongoDB.MetaDB,
			"database_requested": name,
			"_id":                dbr.ID.Hex,
			"error":              err.Error(),
		}).Error("Could not retrieve timestamp range from metadatabase: ", err)
		return min, max, err
	}

	min = tsRes.TSRange.Min
	max = tsRes.TSRange.Max

	return min, max, nil
}

// AddTSRange adds the min and max timestamps found in current dataset
func (m *MetaDB) AddTSRange(name string, min int64, max int64) error {
	dbr, err := m.GetDBMetaInfo(name)

	if err != nil {
		m.log.WithFields(log.Fields{
			"database_requested": name,
			"error":              err.Error(),
		}).Error("Could not add timestamp range: database not found in metadata directory")
		return err
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	_, err = ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).
		Upsert(
			bson.M{"_id": dbr.ID},
			bson.M{
				"$set": bson.M{
					"ts_range.min": min,
					"ts_range.max": max,
				}},
		)

	if err != nil {
		m.log.WithFields(log.Fields{
			"metadb_attempted":   m.config.S.MongoDB.MetaDB,
			"database_requested": name,
			"_id":                dbr.ID.Hex,
			"error":              err.Error(),
		}).Error("Could not update timestamp range for database entry in metadatabase")
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

	_, err = ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).
		Upsert(bson.M{"_id": dbr.ID}, bson.M{
			"$set": bson.D{
				{"analyzed", complete},
				{"analyze_version", versionTag},
			},
		})

	if err != nil {
		m.log.WithFields(log.Fields{
			"metadb_attempted":   m.config.S.MongoDB.MetaDB,
			"database_requested": name,
			"_id":                dbr.ID.Hex,
			"error":              err.Error(),
		}).Error("could not update database entry in meta")
		return err
	}
	return nil
}

// runDBMetaInfoQuery runs a MongoDB query against the MetaDB Databases Table
// and performs any necessary data migration
func (m *MetaDB) runDBMetaInfoQuery(queryDoc bson.M) ([]DBMetaInfo, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	var results []DBMetaInfo
	err := ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).Find(queryDoc).All(&results)
	if err != nil {
		return results, err
	}

	return results, nil
}

// SetChunk ....
func (m *MetaDB) SetChunk(cid int, db string, analyzed bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	_, err := ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).
		Upsert(
			bson.M{"name": db},
			bson.M{
				"$set": bson.M{
					"cid_list." + strconv.Itoa(cid) + ".set": analyzed,
				}},
		)

	if err != nil {
		m.log.WithFields(log.Fields{
			"metadb_attempted":   m.config.S.MongoDB.MetaDB,
			"database_requested": db,
			"error":              err.Error(),
		}).Error("Could not update CID analyzed value for database entry in metadatabase")
		return err
	}
	return nil
}

// IsChunkSet ....
func (m *MetaDB) IsChunkSet(cid int, db string) (bool, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	query := bson.M{
		"$and": []interface{}{
			bson.M{"name": db},
			bson.M{"cid_list." + strconv.Itoa(cid) + ".set": true},
		},
	}

	var results []interface{}
	err := ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).Find(query).All(&results)

	if err != nil {
		return false, err
	}

	if len(results) > 0 {
		return true, nil
	}

	return false, nil
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
		m.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not complete schema migration step")
		return nil
	}

	var results []string
	for _, db := range dbs {
		results = append(results, db.Name)
	}
	return results

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

// GetFiles gets a list of all file hashes for the database
func (m *MetaDB) GetFiles(database string) ([]string, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.dbHandle.Copy()
	defer ssn.Close()

	var hashes []string
	err := ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).
		Find(bson.M{"name": database}).
		Distinct("file_hashes.hash", &hashes)

	// check if files table exists and query for backwards compatibility
	// TODO: remove me on majore version upgrade
	var legacyHashes []string
	err2 := ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.FilesTable).
		Find(bson.M{"database": database}).
		Distinct("hash", &legacyHashes)

	if err == nil && err2 == nil {
		// both were valid, include all hashes
		hashes = append(hashes, legacyHashes...)
	} else if err != nil && err2 == nil {
		// hashes failed so fallback to legacy hashes
		hashes = legacyHashes
		err = nil
	} else if err != nil && err2 != nil {
		// both queries failed, just report first error
		return nil, err
	}

	return hashes, err
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

	db := ssn.DB(m.config.S.MongoDB.MetaDB).C(m.config.T.Meta.DatabasesTable).Bulk()
	db.Unordered()

	for _, file := range files {
		db.Update(
			// selector for database entry
			bson.M{"name": file.TargetDatabase},
			// update that appends new file hash to file_hashes array
			bson.M{
				"$push": bson.M{
					"file_hashes": bson.M{
						"hash": file.Hash,
			            "cid": m.config.S.Rolling.CurrentChunk,
					},
				},
			},
		)
	}

	_, err := db.Run()
	if err != nil {
		m.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("could not insert files into meta database")
		return err
	}
	return nil
}
