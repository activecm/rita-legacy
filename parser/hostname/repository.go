package hostname

// Repository for hostnames collection
type Repository interface {
	CreateIndexes() error
	// Upsert(hostname *parsetypes.Hostname) error
	Upsert(domainMap map[string][]string)
}

//update ....
type update struct {
	selector interface{}
	query    interface{}
}

//hostname ....
type hostname struct {
	name    string
	answers []string
}
