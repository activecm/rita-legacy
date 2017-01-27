package diagnostic

import (
	"sync"
	"time"

	"github.com/ocmdev/rita/config"

	mgo "gopkg.in/mgo.v2"
)

type (
	userController struct {
		session *mgo.Database
	}
	db struct {
		Database *mgo.Database
	}
)

func newUserController(s *mgo.Database) *userController {
	return &userController{s}
}

var initCTX sync.Once
var _instance *db

// LogFail should be called when anything fails, takes a failed log and when it occured
func LogFail(failedLog interface{}, cfg *config.Resources, when time.Time) {
	uc := newUserController(newMongoSession())

	if when.IsZero() {
		when = time.Now()
	}

	loggedError := diagModel{"Error message", when, failedLog}

	if err := uc.session.C("errors").Insert(loggedError); err != nil {
		panic(err)
	}
}

func newMongoSession() *mgo.Database {
	initCTX.Do(func() {
		_instance = new(db)
		session, err := mgo.Dial("mongodb://localhost")
		if err != nil {
			panic(err)
		}
		_instance.Database = session.DB("RITAerrors")
	})

	return _instance.Database
}
