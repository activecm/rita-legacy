package urls

import (
	datatype_Data "github.com/ocmdev/rita/datatypes/data"

	"gopkg.in/mgo.v2/bson"
)

type (
	/*** Collection/Reporting Structure ***/
	Url struct {
		ID     bson.ObjectId `bson:"_id,omitempty"`
		Url    string        `bson:"url"`
		Uri    string        `bson:"uri"`
		IP     string        `bson:"ip"`
		Length int64         `bson:"length"`
		Uid    string        `bson:"uid"`
	}

	/*** Hostname is a mapping from a dns name to ip addresses ***/
	Hostname struct {
		Host string   `bson:"host"`
		IPs  []string `bson:"ips"`
	}

	/*** Graphing Structures ***/
	LenObj struct {
		Length int      `bson:"_id" json:"_id"`
		Src    []string `bson:"src" json:"src"`
		Url    []string `bson:"url" json:"url"`
		Uri    []string `bson:"uri" json:"uri"`
	}

	ConnObj struct {
		Lengths []int   `bson:"lengths" json:"lengths"`
		Longest int     `bson:"longest" json:"longest"`
		Src     string  `bson:"src" json:"src"`
		Url     string  `bson:"url" json:"url"`
		Hits    int     `bson:"connection_count" json:"connection_count"`
		TSS     []int64 `bson:"tss" json:"tss"`
	}

	SrcObj struct {
		UrlList []string `bson:"urls" json:"urls"`
		UrlCt   int      `bson:"url_count" json:"url_count"`
		Lengths [][]int  `bson:"lengths" json:"lengths"`
		Longest []int    `bson:"longest" json:"longest"`
		Src     string   `bson:"src" json:"src"`
		Hits    int      `bson:"connection_count" json:"connection_count"`
	}

	DstObj struct {
		Srcs    []string `bson:"srcs" json:"srcs"`
		SrcCt   int      `bson:"source_count" json:"source_count"`
		Lengths [][]int  `bson:"lengths" json:"lengths"`
		Longest []int    `bson:"longest" json:"longest"`
		Url     string   `bson:"url" json:"url"`
		Hits    int      `bson:"connection_count" json:"connection_count"`
	}

	/*** TBD Summary Object ***/
	SumObj struct {
		SrcCt    int `bson:"unique_src_count" json:"unique_src_count"`
		DstCt    int `bson:"unique_dst_count" json:"unique_dst_count"`
		LengthCt int `bson:"unique_length_count" json:"unique_length_count"`
		Longest  int `bson:"longest" json:"longest"`
		Hits     int `bson:"connection_count" json:"connection_count"`
	}

	PartObj struct {
		Url    string                `bson:"url" json:"url"`
		Uri    string                `bson:"uri" json:"uri"`
		Length int                   `bson:"length" json:"length"`
		Conn   datatype_Data.ConnObj `bson:"conn_join" json:"conn_join"`
	}
)
