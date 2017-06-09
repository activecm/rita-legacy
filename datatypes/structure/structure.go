package structure

import (
	"gopkg.in/mgo.v2/bson"
)

type (
	Host struct {
		ID    bson.ObjectId `bson:"_id,omitempty"`
		Ip    string        `bson:"ip"`
		Local bool          `bson:"local"`
	}

	UniqueConnection struct {
		ID              bson.ObjectId `bson:"_id,omitempty"`
		ConnectionCount int           `bson:"connection_count"`
		Src             string        `bson:"src"`
		Dst             string        `bson:"dst"`
		LocalSrc        bool          `bson:"local_src"`
		LocalDst        bool          `bson:"local_dst"`
		TotalBytes      int           `bson:"total_bytes"`
		OriginBytes     int64         `bson:"origin_bytes"`
		AverageBytes    float32       `bson:"average_bytes"`
		TotalDuration   float32       `bson:"total_duration"`
	}
)
