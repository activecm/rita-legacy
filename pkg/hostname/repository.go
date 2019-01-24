package hostname

import (
	"github.com/activecm/rita/parser/parsetypes"
)

// Repository for hostnames collection
type Repository interface {
	CreateIndexes(targetDB string) error
	Upsert(hostname *parsetypes.Hostname, targetDB string) error
}
