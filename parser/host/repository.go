package host

import("github.com/activecm/rita/parser/parsetypes")

type Repository interface {
	Create(host *Host) error
	Update(host *Host) error
}