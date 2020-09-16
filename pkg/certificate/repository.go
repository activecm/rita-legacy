package certificate

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(useragentMap map[string]*Input)
}

//update ....
type update struct {
	selector   interface{}
	query      interface{}
	collection string
}

//TODO[AGENT]: Convert Input Host to UniqueIP {Host, NetworkID, NetworkName}
//TODO[AGENT]: Convert Input OrigIPs to []UniqueIP
//Input ....
type Input struct {
	Host         string
	Seen         int64
	OrigIps      []string
	InvalidCerts []string
	Tuples       []string
}

//AnalysisView (for reporting)
type AnalysisView struct {
	UserAgent string `bson:"user_agent"`
	TimesUsed int64  `bson:"seen"`
}
