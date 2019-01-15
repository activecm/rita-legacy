package conn

import("github.com/activecm/rita/parser/parsetypes")

type Repository interface {
	BulkDelete(conns []*parsetypes.Conn, targetDB string) error
}