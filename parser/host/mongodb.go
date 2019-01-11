package host

import(
	"github.com/activecm/rita/parser/parsetypes"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

func (fs *FSImporter) Create(host *Host) error {
	resDB := fs.res.DB
	resConf := fs.res.Config
	logger := fs.res.Log

	// create hosts collection
	// Desired indexes
	hostKeys := []mgo.Index{
		{Key: []string{"ip"}, Unique: true},
		{Key: []string{"local"}},
		{Key: []string{"ipv4"}},
		{Key: []string{"ipv4_binary"}},
	}

	errorCheck := resDB.CreateCollection(resConf.T.Structure.HostTable, hostKeys)
		if errorCheck != nil {
		logger.Error("Failed: ", errorCheck)
	}
}

func (fs *FSImporter) Update(host *Host) error {
	resDB := fs.res.DB
	resConf := fs.res.Config
	logger := fs.res.Log

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