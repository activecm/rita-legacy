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
		AverageBytes    float32       `bson:"average_bytes"`
		TotalDuration   float32       `bson:"total_duration"`
	}

	// A complete admin object mirroring database record for all Connections
	Connection struct {
		ID    bson.ObjectId `bson:"_id,omitempty" json:"_id,omitempty"`
		Con   string        `bson:"con" json:"con"`
		Host  string        `bson:"host" json:"host"`
		Dst   string        `bson:"dst" json:"dst"`
		Mods  string        `bson:"mods" json:"mods"`
		MHits int           `bson:"mHits" json:"mHits"`
		HitCt int           `bson:"hitCt" json:"hitCt"`
		ModCt int           `bson:"modCt" json:"modCt"`
		// Times []time.Time   `bson:"tss" json:"tss"`
	}

	Http struct {
		Src string `bson:"id_origin_h"`
	}
)
