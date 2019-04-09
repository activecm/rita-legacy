package useragent

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

//Input ....
type Input struct {
	name     string
	Seen     int64
	OrigIps  []string
	Requests []string
	JA3      bool
}

//AnalysisView (for reporting)
type AnalysisView struct {
	UserAgent string `bson:"user_agent"`
	TimesUsed int64  `bson:"seen"`
}
