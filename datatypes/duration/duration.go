package duration

type (
	/*** Reporting Structure ***/
	Duration struct {
		Src      string  `bson:"id_origin_h" json:"id_origin_h"`
		Dst      string  `bson:"id_resp_h" json:"id_resp_h"`
		Duration float64 `bson:"duration" json:"duration"`
	}

	/*** Graphing Structures ***/
	ConnObj struct {
		Src      string  `bson:"src" json:"src"`
		Dst      string  `bson:"dst" json:"dst"`
		Duration float64 `bson:"duration" json:"duration"`
		TSS      []int64 `bson:"tss" json:"tss"`
	}

	// SrcObj struct {
	// 	Src    string    `bson:"_id" json:"_id"`
	// 	Dsts   []ConnObj `bson:"dsts" json:"dsts"`
	// 	DstCt  int       `bson:"unique_dst_count" json:"unique_dst_count"`
	// 	DurCt  int       `bson:"unique_duration_count" json:"unique_duration_count"`
	// 	Hits   int       `bson:"connection_count" json:"connection_count"`
	// 	TopDur float64   `bson:"longest_duration" json:"longest_duration"`
	// 	BotDur float64   `bson:"shortest_duration" json:"shortest_duration"`
	// }

	// DstObj struct {
	// 	Dst    string    `bson:"_id" json:"_id"`
	// 	Srcs   []ConnObj `bson:"srcs" json:"srcs"`
	// 	SrcCt  int       `bson:"unique_src_count" json:"unique_src_count"`
	// 	DurCt  int       `bson:"unique_duration_count" json:"unique_duration_count"`
	// 	Hits   int       `bson:"connection_count" json:"connection_count"`
	// 	TopDur float64   `bson:"longest_duration" json:"longest_duration"`
	// 	BotDur float64   `bson:"shortest_duration" json:"shortest_duration"`
	// }

	// DurObj struct {
	// 	Duration float64   `bson:"_id" json:"_id"`
	// 	Conn     []ConnObj `bson:"srcs" json:"srcs"`
	// 	SrcCt    int       `bson:"unique_src_count" json:"unique_src_count"`
	// 	DstCt    int       `bson:"unique_dst_count" json:"unique_dst_count"`
	// 	Hits     int       `bson:"connection_count" json:"connection_count"`
	// }

	// /*** Duration Summary Object ***/
	// SumObj struct {
	// 	TopDur float64 `bson:"longest_duration" json:"longest_duration"`
	// 	BotDur float64 `bson:"shortest_duration" json:"shortest_duration"`
	// 	SrcCt  int     `bson:"unique_src_count" json:"unique_src_count"`
	// 	DstCt  int     `bson:"unique_dst_count" json:"unique_dst_count"`
	// 	DurCt  int     `bson:"unique_duration_count" json:"unique_duration_count"`
	// 	Hits   int     `bson:"connection_count" json:"connection_count"`
	// }
)
