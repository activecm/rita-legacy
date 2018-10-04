package database

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"

	"github.com/blang/semver"
)

type (
	// RITADatabase provides methods for manipulating an
	// entry in the RITADatabaseIndex
	RITADatabase struct {
		indexDoc            DBMetaInfo
		metaDatabaseName    string
		indexCollectionName string
		log                 *log.Logger
	}

	// DBMetaInfo defines some information about the database
	DBMetaInfo struct {
		ID             bson.ObjectId `bson:"_id,omitempty"`   // Ident
		Name           string        `bson:"name"`            // Name of the database
		ImportFinished bool          `bson:"import_finished"` // Has this database finished being imported
		Analyzed       bool          `bson:"analyzed"`        // Has this database been analyzed
		ImportVersion  string        `bson:"import_version"`  // Rita version at import
		AnalyzeVersion string        `bson:"analyze_version"` // Rita version at analyze
	}
)

// Name returns the name of the database. Can be used with ssn.DB(_).
func (r *RITADatabase) Name() string {
	return r.indexDoc.Name
}

// ImportFinished returns whether a database is still being imported or not
func (r *RITADatabase) ImportFinished() bool {
	return r.indexDoc.ImportFinished
}

// Analyzed returns whether the database has been analyzed or not
func (r *RITADatabase) Analyzed() bool {
	return r.indexDoc.Analyzed
}

// ImportVersion returns the version of RITA the database was imported with
func (r *RITADatabase) ImportVersion() (semver.Version, error) {
	return semver.ParseTolerant(r.indexDoc.ImportVersion)
}

// AnalyzeVersion returns the version of RITA the database was analyzed with
func (r *RITADatabase) AnalyzeVersion() (semver.Version, error) {
	return semver.ParseTolerant(r.indexDoc.AnalyzeVersion)
}

// CompatibleImportVersion checks if the database was imported
// using a compatible version of RITA
func (r *RITADatabase) CompatibleImportVersion(version semver.Version) (bool, error) {
	importVersion, err := r.ImportVersion()
	if err != nil {
		return false, err
	}
	return version.Major == importVersion.Major, nil
}

// CompatibleAnalyzeVersion checks if the database was analyzed
// using a compatible version of RITA
func (r *RITADatabase) CompatibleAnalyzeVersion(version semver.Version) (bool, error) {
	analyzeVersion, err := r.AnalyzeVersion()
	if err != nil {
		return false, err
	}
	return version.Major == analyzeVersion.Major, nil
}

// SetImportFinished marks this RITADatabase as being completely imported in the RITADatabaseIndex
func (r *RITADatabase) SetImportFinished(ssn *mgo.Session) error {
	err := ssn.DB(r.metaDatabaseName).C(r.indexCollectionName).Update(
		bson.M{"name": r.indexDoc.Name},
		bson.M{
			"$set": bson.M{
				"import_finished": true,
			},
		},
	)

	if err != nil {
		r.log.WithFields(log.Fields{
			"metadb":           r.metaDatabaseName,
			"index_collection": r.indexCollectionName,
			"database":         r.indexDoc.Name,
			"error":            err.Error(),
		}).Error("could not mark database imported in database index")
		return err
	}
	return nil
}

// UnsetImportFinished marks this RITADatabase as not being completely imported in the RITADatabaseIndex
func (r *RITADatabase) UnsetImportFinished(ssn *mgo.Session) error {
	err := ssn.DB(r.metaDatabaseName).C(r.indexCollectionName).Update(
		bson.M{"name": r.indexDoc.Name},
		bson.M{
			"$set": bson.M{
				"import_finished": false,
				"import_version":  "",
			},
		},
	)

	if err != nil {
		r.log.WithFields(log.Fields{
			"metadb":           r.metaDatabaseName,
			"index_collection": r.indexCollectionName,
			"database":         r.indexDoc.Name,
			"error":            err.Error(),
		}).Error("could not mark database as not imported in database index")
		return err
	}
	return nil
}

// SetAnalyzed marks this RITADatabase as analyzed in the RITADatabaseIndex
func (r *RITADatabase) SetAnalyzed(ssn *mgo.Session, ritaVersion semver.Version) error {

	err := ssn.DB(r.metaDatabaseName).C(r.indexCollectionName).Update(
		bson.M{"name": r.indexDoc.Name},
		bson.M{
			"$set": bson.M{
				"analyzed":        true,
				"analyze_version": ritaVersion.String(),
			},
		},
	)

	if err != nil {
		r.log.WithFields(log.Fields{
			"metadb":           r.metaDatabaseName,
			"index_collection": r.indexCollectionName,
			"database":         r.indexDoc.Name,
			"error":            err.Error(),
		}).Error("could not mark database analyzed in database index")
		return err
	}
	return nil
}

// UnsetAnalyzed marks this RITADatabase as unanalyzed in the RITADatabaseIndex
func (r *RITADatabase) UnsetAnalyzed(ssn *mgo.Session) error {

	err := ssn.DB(r.metaDatabaseName).C(r.indexCollectionName).Update(
		bson.M{"name": r.indexDoc.Name},
		bson.M{
			"$set": bson.M{
				"analyzed":        false,
				"analyze_version": "",
			},
		},
	)

	if err != nil {
		r.log.WithFields(log.Fields{
			"metadb":           r.metaDatabaseName,
			"index_collection": r.indexCollectionName,
			"database":         r.indexDoc.Name,
			"error":            err.Error(),
		}).Error("could not mark database as not analyzed in database index")
		return err
	}
	return nil
}

// DeleteIndex removes this database from the RITADatabaseIndex
// Note: This does not drop the database
func (r *RITADatabase) DeleteIndex(ssn *mgo.Session) error {
	return ssn.DB(r.metaDatabaseName).C(r.indexCollectionName).Remove(bson.M{
		"name": r.indexDoc.Name,
	})
}

// Drop drops the referenced database from MongoDB
func (r *RITADatabase) Drop(ssn *mgo.Session) error {
	return ssn.DB(r.indexDoc.Name).DropDatabase()
}
