package structure

import (
	"gopkg.in/mgo.v2/bson"
)

type (
	Host struct {
		Ip    string `bson:"ip"`
		Local bool   `bson:"local"`
	}

	UniqueConnection struct {
		ID              bson.ObjectId `bson:"_id,omitempty"`
		ConnectionCount int           `bson:"connection_count"`
		Src             string        `bson:"src"`
		Dst             string        `bson:"dst"`
		LocalSrc        bool          `bson:"local_src"`
		LocalDst        bool          `bson:"local_dst"`
		TotalBytes      int           `bson:"total_bytes"`
		AverageBytes    float32       `bson:"average_bytes"`
		TotalDuration   float32       `bson:"total_duration"`
	}

	//srcIPGroup holds information used to find the number of unique connections,
	//total connections, and total bytes for a blacklisted url, but are grouped by
	//the ip that connected to the blacklisted url
	SrcIPGroup struct {
		ID         bson.ObjectId `bson:"_id,omitempty"`
		TotalBytes int           `bson:"total_bytes"`
		TotalConns int           `bson:"total_conn"`
	}
)
