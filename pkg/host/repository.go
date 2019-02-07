package host

import "github.com/activecm/rita/pkg/uconn"

// Repository for host collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]uconn.Pair)
}

//update ....
type update struct {
	selector interface{}
	query    interface{}
}
