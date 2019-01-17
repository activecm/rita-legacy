package uconn

import (
	"github.com/activecm/rita/parser/parsetypes"
)

// Repository for uconn collection
type Repository interface {
	CreateIndexes(targetDB string) error
	Insert(uconn *parsetypes.Uconn, targetDB string) error
	Upsert(uconn *parsetypes.Uconn, targetDB string) error
}
