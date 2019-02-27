package remover

// Repository ....
type Repository interface {
	Remove(int) error
}

//update ....
type update struct {
	selector   interface{}
	query      interface{}
	collection string
}

// //Input ....
// type Input struct {
// 	name     string
// 	Seen     int64
// 	OrigIps  []string
// 	Requests []string
// }
