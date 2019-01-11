package uconn

import("github.com/activecm/rita/parser/parsetypes")

type Repository interface {
	Insert(uconn *Uconn) error
}