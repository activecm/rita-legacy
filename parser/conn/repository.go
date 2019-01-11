package conn

import("github.com/activecm/rita/parser/parsetypes")

type Repository interface {
	BulkDeleteSetup(conns []*Conn) (bulk, error)
	BulkDeleteRun(bulk *bulk)
}