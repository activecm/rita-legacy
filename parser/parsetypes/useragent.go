package parsetypes

import (
	"github.com/globalsign/mgo/bson"
)

type (
	// UserAgent stores user agent usage information
	UserAgent struct {
		ID        bson.ObjectId `bson:"_id,omitempty"`
		UserAgent string        `bson:"user_agent"`
		TimesUsed int32         `bson:"times_used"`
	}
)
