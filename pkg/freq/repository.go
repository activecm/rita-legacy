package freq

import (
	"github.com/activecm/rita/parser/parsetypes"
)

// Repository for freq collection
type Repository interface {
	Insert(freqConn *parsetypes.Freq) error
}

//AnalysisView  (for reporting)
type AnalysisView struct {
	Source          string `bson:"src"`
	Destination     string `bson:"dst"`
	ConnectionCount int64  `bson:"connection_count"`
}
