package host

import(
	"github.com/activecm/rita/parser/parsetypes"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
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

func (r *repo) Create(host *Host, targetDB string) error {
	session := r.pool.Session(nil)
	coll := session.DB(targetDB).C("host")

	// create hosts collection
	// Desired indexes
	hostKeys := []mgo.Index{
		{Key: []string{"ip"}, Unique: true},
		{Key: []string{"local"}},
		{Key: []string{"ipv4"}},
		{Key: []string{"ipv4_binary"}},
	}

	err := resDB.CreateCollection(resConf.T.Structure.HostTable, hostKeys)
	if err != nil {
		return err
	}
	return nil
}

func (r *repo) Upsert(host *Host, targetDB string) error {

	// set up update query
	srcQuery := bson.D{
		{"$setOnInsert", bson.M{"local": host.isLocalSrc}},
		{"$setOnInsert", bson.M{"ipv4": srcIP4}},
		{"$inc", bson.M{"count_src": 1}},
		{"$max", bson.M{"max_duration": host.maxDuration}},
	}

	// add ipv4 binary if ipv4
	if srcIP4 {
		srcIPv4bin := ipv4ToBinary(net.ParseIP(host.src))
		srcQuery = append(srcQuery, bson.DocElem{"$setOnInsert", bson.M{"ipv4_binary": srcIPv4bin}})
	} //else{} // future ipv6 support will have ipv6 binary added here

	// update hosts field
	ssn.DB(targetDB).C(resConf.T.Structure.HostTable).Upsert(
		bson.M{"ip": uconnMap[uconn].src},
		srcQuery)

			// **** add uconn dst to hosts table if it doesn't already exist *** //
	// check if ipv4
	dstIP4 := isIPv4(uconnMap[uconn].dst)

	// set up update query
	dstQuery := bson.D{
		{"$setOnInsert", bson.M{"local": uconnMap[uconn].isLocalDst}},
		{"$setOnInsert", bson.M{"ipv4": dstIP4}},
		{"$inc", bson.M{"count_dst": 1}},
		{"$max", bson.M{"max_duration": uconnMap[uconn].maxDuration}},
	}

	// add ipv4 binary if ipv4
	if dstIP4 {
		dstIPv4bin := ipv4ToBinary(net.ParseIP(uconnMap[uconn].dst))
		dstQuery = append(dstQuery, bson.DocElem{"$setOnInsert", bson.M{"ipv4_binary": dstIPv4bin}})
	} //else{} // future ipv6 support will have ipv6 binary added here

	// update hosts field
	ssn.DB(targetDB).C(resConf.T.Structure.HostTable).Upsert(
		bson.M{"ip": uconnMap[uconn].dst},
		dstQuery)
}