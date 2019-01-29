package host

import (
	"github.com/activecm/rita/parser/parsetypes"
)

// Repository for host collection
type Repository interface {
	CreateIndexes() error
	Upsert(host *parsetypes.Host, isSrc bool) error
}
