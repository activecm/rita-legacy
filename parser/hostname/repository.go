package hostname

import (
	"github.com/activecm/rita/parser/parsetypes"
)

// Repository for hostnames collection
type Repository interface {
	CreateIndexes() error
	Upsert(hostname *parsetypes.Hostname) error
}
