package uconn

import("github.com/activecm/rita/parser/parsetypes")

type Repository interface {
	Insert(uconn *parsetypes.Uconn, targetDB string) error
}