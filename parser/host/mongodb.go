package host

import(
	"github.com/juju/mgosession"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/globalsign/mgo"
	"github.com/activecm/rita/resources"
)
type repo struct {
	pool *mgosession.Pool
}

//NewMongoRepository create new repository
func NewMongoRepository(p *mgosession.Pool) Repository {
	return &repo{
		pool: p,
	}
}

func (r *repo) CreateIndexes(targetDB string) error {
	session := r.pool.Session(nil)
	db := session.DB(targetDB)

	// create hosts collection
	// Desired indexes
	hostKeys := []mgo.Index{
		{Key: []string{"ip"}, Unique: true},
		{Key: []string{"local"}},
		{Key: []string{"ipv4"}},
		{Key: []string{"ipv4_binary"}},
	}

	err := db.CreateCollection(resConf.T.Structure.HostTable, hostKeys)
	if err != nil {
		return err
	}
	return nil
}

func (r *repo) Upsert(host *parsetypes.Host, targetDB string) error {
	session := r.pool.Session(nil)
	coll := session.DB(targetDB).C("host")

	// set up update query
	query := bson.D{
		{"$setOnInsert", bson.M{"local": host.isLocalSrc}},
		{"$setOnInsert", bson.M{"ipv4": srcIP4}},
		{"$inc", bson.M{"count_src": 1}},
		{"$max", bson.M{"max_duration": host.maxDuration}},
	}

	// update hosts field
	err := coll.(resConf.T.Structure.HostTable).Upsert(
		bson.M{"ip": host.src},
		query)

	if err != nil {
		return err
	}
	return nil
}