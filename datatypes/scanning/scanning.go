package scanning

import (
	// "gopkg.in/mgo.v2/bson"
	datatype_Data "github.com/bglebrun/rita/datatypes/data"
)

type (
	/*** Reporting Structure ***/
	Scan struct {
		Src          string `bson:"src" json:"src"`
		Dst          string `bson:"dst" json:"dst"`
		PortSetCount int    `bson:"port_count" json:"port_count"`
		PortSet      []int  `bson:"port_set" json:"port_set"`
	}

	/*** Graphing Structures ***/
	ConnObj struct {
		Hits     int       `bson:"connection_count" json:"connection_count"`
		Ports    []int     `bson:"ports" json:"ports"`
		UPorts   []int     `bson:"uports" json:"uports"`
		UPortCt  int       `bson:"uportcount" json:"uportcount"`
		TSS      []int64   `bson:"tss" json:"tss"`
		Duration []float64 `bson:"duration" json:"duration"`
		Src      string    `bson:"src" json:"src"`
		Dst      string    `bson:"dst" json:"dst"`
	}

	SrcObj struct {
		Src   string    `bson:"_id" json:"_id"`
		Dsts  []ConnObj `bson:"scannees" json:"scannees"`
		DstCt int       `bson:"scannee_count" json:"scannee_count"`
		Hits  int       `bson:"total_connection_count" json:"total_connection_count"`
	}

	DstObj struct {
		Dst   string    `bson:"_id" json:"_id"`
		Srcs  []ConnObj `bson:"scanners" json:"scanners"`
		SrcCt int       `bson:"scanner_count" json:"scanner_count"`
		Hits  int       `bson:"total_connection_count" json:"total_connection_count"`
	}

	PortObj struct {
		Port  int       `bson:"_id" json:"_id"`
		Hits  int       `bson:"connection_count" json:"connection_count"`
		Conns []ConnObj `bson:"connections" json:"connections"`
		SrcCt int       `bson:"unique_src_count" json:"unique_src_count"`
		DstCt int       `bson:"unique_dst_count" json:"unique_dst_count"`
	}

	PartObj struct {
		Hits   int                   `bson:"connection_count" json:"connection_count"`
		Ports  []int                 `bson:"port_set" json:"port_set"`
		PortCt int                   `bson:"port_count" json:"port_count"`
		Src    string                `bson:"src" json:"src"`
		Dst    string                `bson:"dst" json:"dst"`
		Conn   datatype_Data.ConnObj `bson:"conn_join" json:"conn_join"`
	}

	/***Scanning Summary Obj***/
	SumObj struct {
		SrcCt  int `bson:"srcs" json:"srcs"`
		DstCt  int `bson:"dsts" json:"dsts"`
		PortCt int `bson:"ports" json:"ports"`
		Hits   int `bson:"hits" json:"hits"`
	}
)
