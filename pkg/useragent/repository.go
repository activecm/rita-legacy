package useragent

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(useragentMap map[string]*Input)
}

type updateInfo struct {
	selector interface{}
	query    interface{}
}

//update ....
type update struct {
	userAgent updateInfo
	host      updateInfo
}

//Input ....
type Input struct {
	name     string
	Seen     int64
	OrigIps  []string
	Requests []string
}

//AnalysisView (for reporting)
type AnalysisView struct {
	UserAgent string `bson:"user_agent"`
	TimesUsed int64  `bson:"times_used"`
}
