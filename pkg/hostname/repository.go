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

type hostname struct {
	host string   `bson:"host"`
	ips  []string `bson:"ips"`
}
