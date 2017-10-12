package beacon

import (
	"gopkg.in/mgo.v2/bson"
)

type (
	//BeaconAnalysisOutput contains the summary statistics of a unique connection
	BeaconAnalysisOutput struct {
		UconnID           bson.ObjectId `bson:"uconn_id"`
		TS_iRange         int64         `bson:"ts_iRange"`
		TS_iMode          int64         `bson:"ts_iMode"`
		TS_iModeCount     int64         `bson:"ts_iMode_count"`
		TS_intervals      []int64       `bson:"ts_intervals"`
		TS_intervalCounts []int64       `bson:"ts_interval_counts"`
		TS_iDispersion    int64         `bson:"ts_iDispersion"`
		TS_iSkew          float64       `bson:"ts_iSkew"`
		TS_duration       float64       `bson:"ts_duration"`
		TS_score          float64       `bson:"ts_score"`
		DS_range          int64         `bson:"ds_range"`
		DS_mode           int64         `bson:"ds_mode"`
		DS_modeCount      int64         `bson:"ds_mode_count"`
		DS_sizes          []int64       `bson:"ds_sizes"`
		DS_sizeCounts     []int64       `bson:"ds_counts"`
		DS_dispersion     int64         `bson:"ds_dispersion"`
		DS_skew           float64       `bson:"ds_skew"`
		DS_score          float64       `bson:"ds_score"`
		Score             float64       `bson:"score"`
	}

	//Used in order to join the uconn and beacon tables
	BeaconAnalysisView struct {
		Src            string  `bson:"src"`
		Dst            string  `bson:"dst"`
		LocalSrc       bool    `bson:"local_src"`
		LocalDst       bool    `bson:"local_dst"`
		Connections    int64   `bson:"connection_count"`
		AvgBytes       float64 `bson:"avg_bytes"`
		TS_iRange      int64   `bson:"ts_iRange"`
		TS_iMode       int64   `bson:"ts_iMode"`
		TS_iModeCount  int64   `bson:"ts_iMode_count"`
		TS_iSkew       float64 `bson:"ts_iSkew"`
		TS_iDispersion int64   `bson:"ts_iDispersion"`
		TS_duration    float64 `bson:"ts_duration"`
		Score          float64 `bson:"score"`
		DS_skew        float64 `bson:"ds_skew"`
		DS_dispersion  int64   `bson:"ds_dispersion"`
		DS_range       int64   `bson:"ds_range"`
		DS_mode        int64   `bson:"ds_mode"`
		DS_modeCount   int64   `bson:"ds_mode_count"`
	}
)
