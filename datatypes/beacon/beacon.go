package beacon

import (
	"gopkg.in/mgo.v2/bson"
)

type (
	//straight output from the beacon analysis
	BeaconAnalysisOutput struct {
		ID                bson.ObjectId `bson:"_id,omitempty"`
		UconnID           bson.ObjectId `bson:"uconn_id"`
		TS_iRange         int64         `bson:"ts_iRange"`
		TS_iMode          int64         `bson:"ts_iMode"`
		TS_iModeCount     int64         `bson:"ts_iMode_count"`
		TS_iSkew          float64       `bson:"ts_iSkew"`
		TS_iDispersion    int64         `bson:"ts_iDispersion"`
		TS_duration       float64       `bson:"ts_duration"`
		TS_score          float64       `bson:"ts_score"`
		TS_intervals      []int64       `bson:"ts_intervals"`
		TS_intervalCounts []int64       `bson:"ts_interval_counts"`
	}

	//Used in order to join the uconn and beacon tables
	BeaconAnalysisView struct {
		ID             bson.ObjectId `bson:"_id,omitempty"`
		Src            string        `bson:"src"`
		Dst            string        `bson:"dst"`
		LocalSrc       bool          `bson:"local_src"`
		LocalDst       bool          `bson:"local_dst"`
		Connections    int64         `bson:"connection_count"`
		AvgBytes       float64       `bson:"avg_bytes"`
		TS_iRange      int64         `bson:"ts_iRange"`
		TS_iMode       int64         `bson:"ts_iMode"`
		TS_iModeCount  int64         `bson:"ts_iMode_count"`
		TS_iSkew       float64       `bson:"ts_iSkew"`
		TS_iDispersion int64         `bson:"ts_iDispersion"`
		TS_duration    float64       `bson:"ts_duration"`
		TS_score       float64       `bson:"ts_score"`
	}
)
