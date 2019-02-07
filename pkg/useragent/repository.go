package useragent

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(useragentMap map[string]*Input)
}

//update ....
type update struct {
	selector interface{}
	query    interface{}
}

type useragent struct {
	name string   `bson:"user_agent"`
	seen int64    `bson:"times_used"`
	ips  []string `bson:"ips"`
}

//Input ....
type Input struct {
	name string
	Seen int64
	Ips  []string
}
