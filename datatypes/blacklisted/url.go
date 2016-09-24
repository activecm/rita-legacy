package blacklisted

type (
	/*** Graphing Structures ***/
	UrlConnObj struct {
		Dst   string  `bson:"host" json:"host"`
		Src   string  `bson:"local_host" json:"local_host"`
		TSS   []int64 `bson:"tss" json:"tss"`
		Score int     `bson:"score" json:"score"`
		Hits  int     `bson:"total_connection_count" json:"total_connection_count"`
	}

	UrlSrcObj struct {
		Src   string       `bson:"_id" json:"_id"`
		Dsts  []UrlConnObj `bson:"blacklist_url" json:"blacklist_url"`
		DstCt int          `bson:"blacklist_url_count" json:"blacklist_url_count"`
		Hits  int          `bson:"total_connections_count" json:"total_connections_count"`
	}

	UrlDstObj struct {
		Dst     string       `bson:"_id" json:"_id"`
		Score   int          `bson:"score" json:"score"`
		Srcs    []UrlConnObj `bson:"local_hosts" json:"local_hosts"`
		Victims int          `bson:"victim_count" json:"victim_count"`
		Hits    int          `bson:"total_connections_count" json:"total_connections_count"`
		SrcCt   int          `bson:"unique_source_count" json:"unique_source_count"`
	}

	/*** Blacklisted URL Summary Object ***/
	UrlSumObj struct {
		SrcCt   int `bson:"unique_src_count" json:"unique_src_count"`
		DstCt   int `bson:"unique_dst_count" json:"unique_dst_count"`
		Victims int `bson:"highest_victim_count" json:"highest_victim_count"`
		Hits    int `bson:"connection_count" json:"connection_count"`
	}
	UrlPartObj struct {
		Dst   string `bson:"host" json:"host"`
		Src   string `bson:"local_host" json:"local_host"`
		TS    int64  `bson:"ts" json:"ts"`
		Score int    `bson:"score" json:"score"`
	}
)
