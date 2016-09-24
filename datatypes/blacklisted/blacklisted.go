package blacklisted

import "gopkg.in/mgo.v2/bson"

type (
	/*** Collection/Reporting Structure ***/
	Blacklist struct {
		ID          bson.ObjectId `bson:"_id,omitempty"`
		BLHash      string        `bson:"bl_hash"`
		Host        string        `bson:"host"`
		Score       int           `bson:"count"`
		DateChecked int64         `bson:"date_checked"`
		BlType      string        `bson:"blacklist_type"`
		IsUrl       bool          `bson:"is_url"`
		IsIp        bool          `bson:"is_ip"`
	}

	/*** Graphing Structures ***/
	IpHostTable struct {
		ID      bson.ObjectId `bson:"_id,omitempty"`
		Host    string        `bson:"host"`
		Score   int           `bson:"score"`
		Sources []struct {
			TimestampList []int64 `bson:"tss"`
			Src           string  `bson:"src"`
		} `bson:"srcs"`
		Destinations []struct {
			TimestampList []int64 `bson:"tss"`
			Dst           string  `bson:"dst"`
		} `bson:"dsts"`
		VictimCount          int `bson:"victim_count"`
		TotalConnectionCount int `bson:"total_connection_count"`
	}

	UrlHostTable struct {
		ID      bson.ObjectId `bson:"_id,omitempty"`
		Host    string        `bson:"host"`
		Score   int           `bson:"score"`
		Sources []struct {
			TimestampList []int64 `bson:"tss"`
			Src           string  `bson:"src"`
		} `bson:"srcs"`
		VictimCount          int `bson:"victim_count"`
		TotalConnectionCount int `bson:"total_connection_count"`
	}
)
