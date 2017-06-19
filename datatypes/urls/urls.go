package urls

//URL represents the results of url analysis
type URL struct {
	URL    string   `bson:"url"`
	URI    string   `bson:"uri"`
	Length int64    `bson:"length"`
	Count  int64    `bson:"count"`
	IPs    []string `bson:"ips"`
}
