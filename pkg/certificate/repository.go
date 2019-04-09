package certificate

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(useragentMap map[string]*Input)
}

type updateInfo struct {
	selector interface{}
	query    interface{}
}

//
// //update ....
// type update struct {
// 	userAgent updateInfo
// 	host      updateInfo
// }

//update ....
type update struct {
	selector   interface{}
	query      interface{}
	collection string
}

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
