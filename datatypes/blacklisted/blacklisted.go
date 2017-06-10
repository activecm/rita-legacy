package blacklisted

import "gopkg.in/mgo.v2/bson"

type (
	/*** Collection/Reporting Structure ***/
	Blacklist struct {
		ID              bson.ObjectId `bson:"_id,omitempty"`
		BLHash          string        `bson:"bl_hash"`
		Host            string        `bson:"host"`
		Score           int           `bson:"count"`
		DateChecked     int64         `bson:"date_checked"`
		BlType          string        `bson:"blacklist_type"`
		IsURL           bool          `bson:"is_url"`
		IsIp            bool          `bson:"is_ip"`
		BlacklistSource []string      `bson:"blacklist_sources"`
		Sources         []string      `bson:"sources,omitempty"`
	}
)
