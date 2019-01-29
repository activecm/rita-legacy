package uconn

import (
	"github.com/activecm/rita/parser/parsetypes"
)

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Insert(uconn *parsetypes.Uconn) error
	Upsert(uconn *parsetypes.Uconn) error
}
