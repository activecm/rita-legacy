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
		Score             float64       `bson:"score"`
		TS_intervals      []int64       `bson:"ts_intervals"`
		TS_intervalCounts []int64       `bson:"ts_interval_counts"`
		DS_skew           float64       `bson:"ds_skew"`
		DS_dispersion     int64         `bson:"ds_dispersion"`
		DS_range          int64         `bson:"ds_range"`
		DS_sizes          []int64       `bson:"ds_sizes"`
		DS_counts         []int64       `bson:"ds_counts"`
		DS_mode           int64         `bson:"ds_mode"`
		DS_modeCount      int64         `bson:"ds_mode_count"`
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
		Score          float64       `bson:"score"`
		DS_skew        float64       `bson:"ds_skew"`
		DS_dispersion  int64         `bson:"ds_dispersion"`
		DS_range       int64         `bson:"ds_range"`
		DS_mode        int64         `bson:"ds_mode"`
		DS_modeCount   int64         `bson:"ds_mode_count"`
	}
)
