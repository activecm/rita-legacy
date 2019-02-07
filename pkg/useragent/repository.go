package useragent

import (
	"github.com/activecm/rita/parser/parsetypes"
)

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(userAgent *parsetypes.UserAgent) error
}
