package freqConn

import("github.com/activecm/rita/parser/parsetypes")

type Repository interface {
	Insert(freqConn *freqConn, datastore) error
}