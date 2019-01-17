package host

import(
	"github.com/activecm/rita/parser/parsetypes"
)

type Repository interface {
	CreateIndexes(targetDB string) error
	Upsert(host *parsetypes.Host, isSrc bool, targetDB string) error
}