package conn

import (
	"github.com/activecm/rita/parser/parsetypes"
)

// Repository for conn collection
type Repository interface {
	BulkDelete(conns []*parsetypes.Conn) error
}
