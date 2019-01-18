package dns

import (
	//"github.com/activecm/rita/parser/parsetypes"
)

// Repository for uconn collection
type Repository interface {
	CreateIndexes(targetDB string) error
	Insert(targetDB string) error
}
