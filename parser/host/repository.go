package host

import("github.com/activecm/rita/parser/parsetypes")

type Repository interface {
	Create(host *Host, targetDB string) error
	Upsert(host *Host, targetDB string) error
}