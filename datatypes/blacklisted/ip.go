package blacklisted

type (

	/*** Graphing Structures ***/
	IpConnObj struct {
		Dst   string  `bson:"host" json:"host"`
		Src   string  `bson:"local_host" json:"local_host"`
		TSS   []int64 `bson:"tss" json:"tss"`
		Score int     `bson:"score" json:"score"`
		Hits  int     `bson:"total_connection_count" json:"total_connection_count"`
	}

	IpSrcObj struct {
		Src   string      `bson:"_id" json:"_id"`
		Dsts  []IpConnObj `bson:"blacklist_ip" json:"blacklist_ip"`
		DstCt int         `bson:"blacklist_ip_count" json:"blacklist_ip_count"`
		Hits  int         `bson:"total_connection_count" json:"total_connection_count"`
	}

	IpDstObj struct {
		Dst     string      `bson:"host" json:"host"`
		Score   int         `bson:"score" json:"score"`
		Srcs    []IpConnObj `bson:"local_hosts" json:"local_hosts"`
		Victims int         `bson:"victim_count" json:"victim_count"`
		Hits    int         `bson:"total_connection_count" json:"total_connection_count"`
		SrcCt   int         `bson:"unique_source_count" json:"unique_source_count"`
	}

	/*** Blacklisted IP Summary Object ***/
	IpSumObj struct {
		SrcCt   int `bson:"unique_src_count" json:"unique_src_count"`
		DstCt   int `bson:"unique_dst_count" json:"unique_dst_count"`
		Victims int `bson:"highest_victim_count" json:"highest_victim_count"`
		Hits    int `bson:"connection_count" json:"connection_count"`
	}
	IpPartObj struct {
		Dst   string `bson:"host" json:"host"`
		Src   string `bson:"local_host" json:"local_host"`
		TS    int64  `bson:"ts" json:"ts"`
		Score int    `bson:"score" json:"score"`
	}
)
