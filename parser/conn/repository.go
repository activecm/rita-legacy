package conn

import("github.com/activecm/rita/parser/parsetypes")

type Repository interface {
	BulkDelete(conns []*Conn, targetDB string) error
}