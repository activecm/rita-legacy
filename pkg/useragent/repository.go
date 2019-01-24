package useragent

import (
	"github.com/activecm/rita/parser/parsetypes"
)

// Repository for uconn collection
type Repository interface {
	CreateIndexes(targetDB string) error
	Upsert(userAgent *parsetypes.UserAgent, targetDB string) error
}
