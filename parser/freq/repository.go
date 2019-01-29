package freq

import (
	"github.com/activecm/rita/parser/parsetypes"
)

// Repository for freq collection
type Repository interface {
	Insert(freqConn *parsetypes.Freq) error
}
