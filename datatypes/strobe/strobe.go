package strobe

//Strobe holds the results of the user agent analysis
type Strobe struct {
	Source          string `bson:"src"`
	Destination     string `bson:"dst"`
	ConnectionCount int64  `bson:"connection_count"`
}
