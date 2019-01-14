package freq

import("github.com/activecm/rita/parser/parsetypes")

type Repository interface {
	Insert(freqConn *parsetypes.Freq, targetDB string) error
}