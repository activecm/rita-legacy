package database

import (
	"fmt"

	"github.com/activecm/mgosec"
	"github.com/activecm/rita/config"
	"github.com/blang/semver"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

//MinMongoDBVersion is the lower, inclusive bound on the
//versions of MongoDB compatible with RITA
var MinMongoDBVersion = semver.Version{
	Major: 4,
	Minor: 2,
	Patch: 0,
}

//MaxMongoDBVersion is the upper, exclusive bound on the
//versions of MongoDB compatible with RITA
var MaxMongoDBVersion = semver.Version{
	Major: 4,
	Minor: 3,
	Patch: 0,
}

// DB is the workhorse container for messing with the database
type DB struct {
	Session  *mgo.Session
	log      *log.Logger
	selected string
}

//NewDB constructs a new DB struct
func NewDB(conf *config.Config, log *log.Logger) (*DB, error) {
	// Jump into the requested database
	session, err := connectToMongoDB(conf, log)
	if err != nil {
		return nil, err
	}
	session.SetSocketTimeout(conf.S.MongoDB.SocketTimeout)
	session.SetSyncTimeout(conf.S.MongoDB.SocketTimeout)
	session.SetCursorTimeout(0)

	return &DB{
		Session:  session,
		log:      log,
		selected: "",
	}, nil
}

//connectToMongoDB connects to MongoDB possibly with authentication and TLS
func connectToMongoDB(conf *config.Config, logger *log.Logger) (*mgo.Session, error) {
	connString := conf.S.MongoDB.ConnectionString
	authMechanism := conf.R.MongoDB.AuthMechanismParsed
	tlsConfig := conf.R.MongoDB.TLS.TLSConfig

	var sess *mgo.Session
	var err error
	if conf.S.MongoDB.TLS.Enabled {
		sess, err = mgosec.Dial(connString, authMechanism, tlsConfig)
	} else {
		sess, err = mgosec.DialInsecure(connString, authMechanism)
	}
	if err != nil {
		return sess, err
	}

	buildInfo, err := sess.BuildInfo()
	if err != nil {
		sess.Close()
		return nil, err
	}

	semVersion, err := semver.ParseTolerant(buildInfo.Version)
	if err != nil {
		sess.Close()
		return nil, err
	}

	if !(semVersion.GE(MinMongoDBVersion) && semVersion.LT(MaxMongoDBVersion)) {
		sess.Close()
		return nil, fmt.Errorf(
			"unsupported version of MongoDB. %s not within [%s, %s)",
			semVersion.String(),
			MinMongoDBVersion.String(),
			MaxMongoDBVersion.String(),
		)
	}

	return sess, nil

}

//SelectDB selects a database for analysis
func (d *DB) SelectDB(db string) {
	d.selected = db
}

//GetSelectedDB retrieves the currently selected database for analysis
func (d *DB) GetSelectedDB() string {
	return d.selected
}

//CollectionExists returns true if collection exists in the currently
//selected database
func (d *DB) CollectionExists(table string) bool {
	ssn := d.Session.Copy()
	defer ssn.Close()
	coll, err := ssn.DB(d.selected).CollectionNames()
	if err != nil {
		d.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Failed collection name lookup")
		return false
	}
	for _, name := range coll {
		if name == table {
			return true
		}
	}
	return false
}

//CreateCollection creates a new collection in the currently selected
//database with the required indexes
func (d *DB) CreateCollection(name string, indexes []mgo.Index) error {
	// Make a copy of the current session
	session := d.Session.Copy()
	defer session.Close()

	d.log.Debug("Building collection: ", name)

	// Create new collection by referencing to it, no need to call Create
	err := session.DB(d.selected).C(name).Create(
		&mgo.CollectionInfo{},
	)

	// Make sure it actually got created
	if err != nil {
		return err
	}

	collection := session.DB(d.selected).C(name)
	for _, index := range indexes {
		err := collection.EnsureIndex(index)
		if err != nil {
			return err
		}
	}

	return nil
}

//AggregateCollection builds a collection via a MongoDB pipeline
func (d *DB) AggregateCollection(sourceCollection string,
	session *mgo.Session, pipeline []bson.D) *mgo.Iter {

	// Identify the source collection we will aggregate information from into the new collection
	if !d.CollectionExists(sourceCollection) {
		d.log.Warning("Failed aggregation: (Source collection: ",
			sourceCollection, " doesn't exist)")
		return nil
	}
	collection := session.DB(d.selected).C(sourceCollection)

	// Create the pipe
	pipe := collection.Pipe(pipeline).AllowDiskUse()

	iter := pipe.Iter()

	// If error, Throw computer against wall and drink 2 angry beers while
	// questioning your life, purpose, and relationships.
	if iter.Err() != nil {
		d.log.WithFields(log.Fields{
			"error": iter.Err().Error(),
		}).Error("Failed aggregate operation")
		return nil
	}
	return iter
}

// MergeBSONMaps recursively merges several bson.M objects into a single map.
// When merging slices of maps with the same associated key, the slices are concatenated.
// If two or more maps define the same key and they are not both bson.M objects,
// a panic occurs. It should be known ahead of time whether keys will conflict
// before calling this function.
func MergeBSONMaps(maps ...bson.M) bson.M {
	result := bson.M{}
	for _, mapToMerge := range maps {
		for keyToMerge, valueToMerge := range mapToMerge {
			// handle new keys
			currVal, currValExists := result[keyToMerge]
			if !currValExists {
				result[keyToMerge] = valueToMerge
				continue
			}

			// handle merging child maps
			currValMap, currValIsMap := currVal.(bson.M)
			mapToMerge, valueToMergeIsMap := valueToMerge.(bson.M)
			if currValIsMap && valueToMergeIsMap {
				result[keyToMerge] = MergeBSONMaps(currValMap, mapToMerge)
				continue
			}

			// handle merging arrays of maps
			currValMapSlice, currValIsMapSlice := currVal.([]bson.M)
			mapSliceToMerge, valueToMergeIsMapSlice := valueToMerge.([]bson.M)
			if currValIsMapSlice && valueToMergeIsMapSlice {
				result[keyToMerge] = append(currValMapSlice, mapSliceToMerge...)
				continue
			}

			// maps cannot be merged due to a type mismatch or overwriting issue
			panic(fmt.Sprintf(
				"BSON maps could not be merged due to conflicting key value pairs:\n"+
					"\tKey: %s\n\tValue 1: %s\n\tValue 2: %s\n",
				keyToMerge, currVal, valueToMerge,
			))
			//return nil
		}
	}

	return result
}
