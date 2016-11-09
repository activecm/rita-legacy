package useragent

import (
	// "gopkg.in/mgo.v2/bson"
	datatype_Data "github.com/bglebrun/rita/datatypes/data"
)

type (
	/*** Reporting Structure ***/
	Useragent struct {
		UseragentString string `bson:"user_agent"`
		TimesUsed       int    `bson:"times_used"`
		Uid             string `bson:"uid"`
	}

	/*** Graphing Structures ***/
	AgentConnObj struct {
		Dsts []string `bson:"dsts" json:"dsts"`
		//destinations accessed by source while using user agent
		TSS   []int64 `bson:"tss" json:"tss"`
		DstCt int     `bson:"unique_dst_count" json:"unique_dst_count"`
		Agent string  `bson:"user_agent" json:"user_agent"`
		Src   string  `bson:"src" json:"src"`
		Hits  int     `bson:"connection_count" json:"connection_count"`
	}

	AgentAgentObj struct {
		Agent string         `bson:"_id" json:"_id"`
		Srcs  []AgentConnObj `bson:"srcs" json:"srcs"`
		SrcCt int            `bson:"unique_source_count" json:"unique_source_count"`
		Hits  int            `bson:"total_connection_count" json:"total_connection_count"`
		//total destinations acessed while this agent was being used
		DstCt int `bson:"total_unique_dsts_count" json:"total_unique_dsts_count"`
	}

	AgentSrcObj struct {
		Src     string         `bson:"_id" json:"_id"`
		Agents  []AgentConnObj `bson:"user_agents" json:"user_agents"`
		AgentCt int            `bson:"unique_agent_count" json:"unique_agent_count"`
		Hits    int            `bson:"total_connection_count" json:"total_connection_count"`
		//total destinations this source reached
		DstCt int `bson:"total_unique_dst_count" json:"total_unique_dst_count"`
	}

	AgentSumObj struct {
		SrcCt    int    `bson:"unique_src_count" json:"srcCt"`
		DstCt    int    `bson:"unique_dst_count" json:"dstCt"`
		ConnCt   int    `bson:"unique_connection_count" json:"connCt"`
		AgentCt  int    `bson:"unique_agent_count" json:"agentCt"`
		Hits     int    `bson:"connection_count" json:"hits"`
		TopSrc   string `bson:"most_used_src" json:"topSrc"`
		TopAgent string `bson:"most_used_agent" json:"topAgent"`
		BotAgent string `bson:"least_used_agent" json:"botAgent"`
		TopConn  string `bson:"most_used_conn" json:"topConn"`
	}

	PartObj struct {
		Agent string                `bson:"user_agent" json:"user_agent"`
		Used  int                   `bson:"times_used" json:"times_used"`
		Conn  datatype_Data.ConnObj `bson:"conn_join" json:"conn_join"`
	}
)
