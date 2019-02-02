package explodeddns

// Repository for explodedDNS collection
type Repository interface {
	CreateIndexes() error
	// Upsert(explodedDNS *parsetypes.ExplodedDNS) error
	Upsert(domainMap map[string]int)
}

//update ....
type update struct {
	selector interface{}
	query    interface{}
}

//domain ....
type domain struct {
	name  string
	count int
}
