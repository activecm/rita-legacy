package inteldb

import (
	"errors"
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/datatypes/intel"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type (
	// IntelDBDocument provides abstraction for the IntelDB data layer
	IntelDBDocument struct {
		// ID holds the database ID for this document
		ID bson.ObjectId `bson:"_id,omitempty"`

		// Host is the IP this document pertains to
		Host string `bson:"host"`

		// Intelligence holds data obtained from whois as well as geo
		// data about this host
		Intelligence data.IntelData `bson:"intel"`

		// BlacklistScore contains score info from the blacklist module
		BlacklistScore int `bson:"blacklist_score"`

		// BlaclistChecked is the date that this item was last checked
		BlacklistChecked time.Time `bson:"blacklist_date_checked"`
	}

	// IntelDBHandle provides the application abstractions for IntelDB
	IntelDBHandle struct {
		// Current configuration
		conf *config.Resources

		// Logger
		log *log.Logger

		// Database that holds the intel data
		db string

		// ssn is the mongo session for the database
		ssn *mgo.Session

		// wchan presents a channel to the writeloop
		wchan chan IntelDBDocument

		// waitGroup gives a way block until the writes are finished
		// after the handle object has been closed
		waitGroup *sync.WaitGroup

		// lock presents locking so we can deconflict write and find
		lock *sync.Mutex
	}

	// Query provides some methods for grabbing data out of the DB and inspecting
	// it. This also implies some methods for existance check.
	Query struct {

		// doc holds the query result
		doc IntelDBDocument

		// ssn to the database for updates
		ssn *mgo.Session

		// if there was a lookup error it will be here
		Error error

		// include lock and unlock so we can work with the db for updates
		dbLock   func()
		dbUnlock func()

		// db gives access to the db string from the parent object
		db string
	}
)

// NewIntelDBHandle provides a new handle to the intelligence database
func NewIntelDBHandle(conf *config.Resources) *IntelDBHandle {
	ssn := conf.CopySession()
	defer ssn.Close()

	// Note that errors in bringing up the database will cause panic
	names, err := ssn.DatabaseNames()
	if err != nil {
		panic(err)
	}

	found := false
	for _, name := range names {
		if name == conf.System.HostIntelDB {
			found = true
		}
	}

	if !found {
		collinfo := mgo.CollectionInfo{
			Capped: false,
		}
		
		//TODO: Use config file for collection name
		err := ssn.DB(conf.System.HostIntelDB).C("external").Create(&collinfo)

		if err != nil {
			panic(err)
		}

		idx := mgo.Index{
			Key:        []string{"host"},
			Unique:     true,
			DropDups:   true,
			Background: true,
		}

		//TODO: Use config file for collection name
		err = ssn.DB(conf.System.HostIntelDB).C("external").EnsureIndex(idx)

		if err != nil {
			panic(err)
		}
	}

	handle := &IntelDBHandle{
		conf:      conf,
		log:       conf.Log,
		db:        conf.System.HostIntelDB,
		ssn:       conf.CopySession(),
		wchan:     make(chan IntelDBDocument),
		waitGroup: new(sync.WaitGroup),
		lock:      new(sync.Mutex),
	}

	go handle.startWriteLoop()

	return handle
}

// Write places an object in the database
func (i IntelDBHandle) Write(dat data.IntelData) {
	ssn := i.ssn.Copy()
	defer ssn.Close()

	dat.IntelDate = time.Now()
	doc := IntelDBDocument{
		Host:             dat.IP,
		Intelligence:     dat,
		BlacklistScore:   -1,
		BlacklistChecked: time.Unix(-1, 0),
	}

	i.wchan <- doc

}

// Find looks up a document by host
func (i IntelDBHandle) Find(host string) *Query {
	ssn := i.ssn.Copy()
	defer ssn.Close()

	var ret IntelDBDocument
	query := &Query{}

	i.lock.Lock()

	//TODO: Use config file for collection name
	// lookup our host
	err := ssn.DB(i.db).C("external").
		Find(bson.M{"host": host}).
		One(&ret)

	i.lock.Unlock()

	// if there was an error mark it so in the query
	if err != nil {
		query.Error = err
	} else {
		query.doc = ret
		query.Error = nil
		query.ssn = i.ssn.Copy()
		query.dbLock = i.lock.Lock
		query.dbUnlock = i.lock.Unlock
		query.db = i.db
	}

	return query

}

// IntelData gives the intel data object that's in the query. Also returns error
// which is the content of the query's error field.
func (q *Query) IntelData(dat *data.IntelData) error {
	*dat = q.doc.Intelligence
	return q.Error
}

// SetBlacklistedScore is used to set the blacklisted score of a particular host
// note that this function will also set the score for the date it was last set
func (q *Query) SetBlacklistedScore(score int) error {

	// if there was an error finding the document just return that
	if q.Error != nil {
		return q.Error
	}

	// create a change object containing the updated information
	change := mgo.Change{
		Update: bson.M{"$set": bson.M{
			"blacklist_score":        score,
			"blacklist_date_checked": time.Now(),
		},
		},
		Upsert:    false,
		Remove:    false,
		ReturnNew: true,
	}

	ssn := q.ssn.Copy()
	defer ssn.Close()

	q.dbLock()
	defer q.dbUnlock()

	var res IntelDBDocument
	//TODO: Use config file for collection name
	info, err := ssn.DB(q.db).C("external").Find(bson.M{"_id": q.doc.ID}).Apply(change, &res)

	// series of sanity checks with the database
	// it is on the programmer using this API to ensure that logging for these things
	// is verbose and accurate to the context in which the error occured
	if err != nil {
		return err
	}

	if info.Updated < 1 {
		return errors.New("Update did not return error, but results appear incorrect")
	}

	if res.BlacklistScore != score {
		return errors.New("Update did not return error, but results are inconsistent")
	}

	if time.Since(res.BlacklistChecked) > (4 * time.Second) {
		return errors.New("blacklisted time checked may be off")
	}

	return nil
}

// GetBlacklistedScore will return a score for an intel entry
func (q *Query) GetBlacklistedScore() (int, error) {

	if q.Error != nil {
		return -2, q.Error
	}

	return q.doc.BlacklistScore, nil

}

// Close shuts down the write loop and blocks until the final writes are complete
func (i IntelDBHandle) Close() {
	close(i.wchan)
	i.waitGroup.Wait()
}

// startWriteLoop spins up a loop that consumes the objects in
// wchan and feeds them to the database
func (i IntelDBHandle) startWriteLoop() {
	ssn := i.ssn.Copy()
	defer ssn.Close()

	i.waitGroup.Add(1)
	for {
		dat, ok := <-i.wchan
		if !ok {
			i.waitGroup.Done()
			return
		}
		i.lock.Lock()
		//TODO: Use config file for collection name
		err := ssn.DB(i.db).C("external").Insert(dat)
		if err != nil {
			i.log.WithFields(log.Fields{
				"error": err.Error(),
				"host":  dat.Host,
			}).Error("insert error in writeloop")
		}
		i.lock.Unlock()
	}
}
