package explodeddns

// Repository for explodedDNS collection
type Repository interface {
	CreateIndexes() error
	// Upsert(explodedDNS *parsetypes.ExplodedDNS) error
	Upsert(domainMap map[string]int)
}

// domain ....
type domain struct {
	name  string
	count int
}

// dns ....
type dns struct {
	Domain         string `bson:"domain"`
	SubdomainCount int64  `bson:"subdomain_count"`
	CID            int    `bson:"cid"`
}

// Result represents a hostname, how many subdomains were found
// for that hostname, and how many times that hostname and its subdomains
// were looked up.
type Result struct {
	Domain         string `bson:"domain"`
	SubdomainCount int64  `bson:"subdomain_count"`
	Visited        int64  `bson:"visited"`
}
