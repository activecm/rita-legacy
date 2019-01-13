package conn

import("github.com/activecm/rita/parser/parsetypes")

type Repository interface {
	BulkDeleteSetup(conns []*Conn, targetDB string) error
}