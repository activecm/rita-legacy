package TBD

import (
	"gopkg.in/mgo.v2/bson"
)

type (

	/*** Layer 1 Collection Structures ***/
	TBD struct {
		ID            bson.ObjectId `bson:"_id,omitempty"`
		Src           string        `bson:"src"` // Going to remove these soon...
		Dst           string        `bson:"dst"` // Going to remove these soon...
		UconnID       bson.ObjectId `bson:"uconn_id"`
		Range         int64         `bson:"range"`
		Size          int64         `bson:"size"`
		RangeVals     string        `bson:"range_vals"`
		Fill          float64       `bson:"fill"`
		Spread        float64       `bson:"spread"`
		Sum           int64         `bson:"range_size"`
		Score         float64       `bson:"score"`
		Intervals     []int64       `bson:"intervals"`
		InvervalCount []int64       `bson:"interval_counts"`
		Tss           []int64       `bson:"tss"`
		TopInterval   int64         `bson:"most_frequent_interval"`
		TopIntervalCt int64         `bson:"most_frequent_interval_count"`
	}

	TBDInput struct {
		ID       bson.ObjectId `bson:"_id,omitempty"`
		Ts       []int64       `bson:"tss"`
		Src      string        `bson:"src"`
		Dst      string        `bson:"dst"`
		LocalSrc bool          `bson:"local_src"`
		LocalDst bool          `bson:"local_dst"`
		Dpts     []int         `bson:"dst_ports"`
		Dur      []float64     `bson:"duration"`
		Count    int           `bson:"connection_count"`
		Bytes    int64         `bson:"total_bytes"`
		BytesAvg float64       `bson:"avg_bytes"`
		Uid      []string      `bson:"uid"`
	}

	/*** Graphing Structures ***/
	ConnObj struct {
		Src           string  `bson:"src" json:"src"`
		Dst           string  `bson:"dst" json:"dst"`
		TSS           []int64 `bson:"tss" json:"tss"`
		Intervals     []int64 `bson:"intervals" json:"intervals"`
		IntervalCount []int   `bson:"interval_counts" json:"interval_counts"`
		TopInterval   int64   `bson:"most_frequent_interval" json:"most_frequent_interval"`
		TopIntervalCt int     `bson:"most_frequent_interval_count" json:"most_frequent_interval_count"`
		Hits          int     `bson:"connection_count" json:"connection_count"`
		Score         float64 `bson:"score"`
	}

	SrcObj struct {
		Src           string    `bson:"_id" json:"_id"`
		Dsts          []ConnObj `bson:"dsts" json:"dsts"`
		DstCt         int       `bson:"unique_dst_count" json:"unique_dst_count"`
		Hits          int       `bson:"connection_count" json:"connection_count"`
		TopInterval   int64     `bson:"most_frequent_interval" json:"most_frequent_interval"`
		TopIntervalCt int       `bson:"most_frequent_interval_count" json:"most_frequent_interval_count"`
	}

	DstObj struct {
		Dst           string    `bson:"_id" json:"_id"`
		Srcs          []ConnObj `bson:"srcs" json:"srcs"`
		SrcCt         int       `bson:"unique_src_count" json:"unique_src_count"`
		Hits          int       `bson:"connection_count" json:"connection_count"`
		TopInterval   int64     `bson:"most_frequent_interval" json:"most_frequent_interval"`
		TopIntervalCt int       `bson:"most_frequent_interval_count" json:"most_frequent_interval_count"`
	}

	IntervalObj struct {
		Interval int64     `bson:"_id" json:"_id"`
		Conns    []ConnObj `bson:"connections" json:"connections"`
		SrcCt    int       `bson:"unique_src_count" json:"unique_src_count"`
		DstCt    int       `bson:"unique_dst_count" json:"unique_dst_count"`
		ConnCt   int       `bson:"unique_connection_count" json:"unique_connection_count"`
		Hits     int       `bson:"connection_count" json:"connection_count"`
	}

	/*** TBD Summary Object ***/
	SumObj struct {
		SrcCt          int     `bson:"unique_src_count" json:"unique_src_count"`
		DstCt          int     `bson:"unique_dst_count" json:"unique_dst_count"`
		IntervalCt     int     `bson:"unique_interval_count" json:"unique_interval_count"`
		Hits           int     `bson:"connection_count" json:"connection_count"`
		TopScore       float64 `bson:"highest_score" json:"highest_score"`
		TopFrequency   int64   `bson:"top_frequency" json:"top_frequency"`
		TopFrequencyCt int     `bson:"top_frequency_count" json:"top_frequency_count"`
	}
)
