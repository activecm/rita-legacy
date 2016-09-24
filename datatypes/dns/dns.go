package dns

type (
	/*** Reporting Structure ***/
	Dns struct {
		QType           string `bson:"qtype_name"`
		ConnectionCount int    `bson:"count"`
		Src             string `bson:"id_origin_h"`
	}

	/*** Graphing Structures ***/
	ConnObj struct {
		Src   string   `bson:"src" json: "src"`
		Dsts  []string `bson:"dsts" json: "dsts"`
		DstCt int      `bson:"unique_dst_count" json:"unique_dst_count"`
		QType string   `bson:"qtype" json:"qtypes"`
		Hits  int      `bson:"connection_count"` // times src made a connection using qtype
		TSS   []int64  `bson:"tss"`
	}

	SrcObj struct {
		Src     string    `bson:"src" json:"src"`
		QTypes  []ConnObj `bson:"dsts" json:"dsts"`
		QTypeCt int       `bson:"unique_qtype_count" json:"unique_qtype_count"`
		Hits    int       `bson:"total_connection_count" json:"total_connection_count"`
		DstCt   int       `bson:"total_unique_dst_count" json:"total_unique_dst_count"` // total dsts this source reached
	}

	QueryObj struct {
		QType string    `bson:"qtype" json:"qtype"`
		Srcs  []ConnObj `bson:"srcs" json:"srcs"`
		SrcCt int       `bson:"unique_src_count" json:"unique_src_count"`
		Hits  int       `bson:"total_connection_count" json:"total_connection_count"`
		DstCt int       `bson:"total_unique_dsts_count" json:"total_unique_dsts_count"` // total dsts acessed while qtype was being used
	}

	/*** DNS Summary Object ***/
	SumObj struct {
		SrcCt   int `bson:"unique_src_count" json:"unique_src_count"`
		DstCt   int `bson:"unique_dst_count" json:"unique_dst_count"`
		ConnCt  int `bson:"unique_connection_count" json:"unique_connection_count"`
		QtypeCt int `bson:"unique_qtype_count" json:"unique_qtype_count"`
		Hits    int `bson:"connection_count" json:"connection_count"`
	}

	/*** Server Collection Structure ***/
	QueryType struct {
		Name         string `bson:"name" json:"name"`
		Abbreviation string `bson:"abrv" json:"abrv"`
		Description  string `bson:"desc" json:"desc"`
	}
	PartObj struct {
		Src   string `bson:"id_origin_h" json: "id_origin_h"`
		Dst   string `bson:"id_resp_h" json: "id_resp_h"`
		QType string `bson:"qtype_name" json:"qtype_name"`
		TS    int64  `bson:"ts" json:"ts"`
	}
)
