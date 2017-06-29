package dns

type (
	//ExplodedDNS maps to an entry in the exploded dns collection
	ExplodedDNS struct {
		Domain     string        `bson:"domain"`
		Subdomains int64         `bson:"subdomains"`
		Visited    int64         `bson:"visited"`
	}

	//Hostname maps to an entry in the hostnames collection
	Hostname struct {
		Host string   `bson:"host"`
		IPs  []string `bson:"ips"`
	}
)
