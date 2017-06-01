package fileparsetypes

import (
	"time"

	pt "github.com/ocmdev/rita/parser/parsetypes"
	"gopkg.in/mgo.v2/bson"
)

//BroHeader contains the parse information contained within the comment lines
//of bro files
type BroHeader struct {
	Names     []string // Names of fields
	Types     []string // Types of fields
	Separator string   // Field separator
	SetSep    string   // Set separator
	Empty     string   // Empty field tag
	Unset     string   // Unset field tag
	ObjType   string   // Object type (comes from #path)
}

//BroHeaderIndexMap maps the names of bro fields to their indexes in a
//BroData struct
type BroHeaderIndexMap map[string]int

//IndexedFile ties a file to a target collection and database
type IndexedFile struct {
	ID               bson.ObjectId `bson:"_id,omitempty"`
	Path             string        `bson:"filepath"`
	Length           int64         `bson:"length"`
	ModTime          time.Time     `bson:"modified"`
	Hash             string        `bson:"hash"`
	TargetCollection string        `bson:"collection"`
	TargetDatabase   string        `bson:"database"`
	LogTime          time.Time     `bson:"date"`
	ParseTime        time.Time     `bson:"time_complete"`
	header           *BroHeader
	broDataFactory   func() pt.BroData
	fieldMap         BroHeaderIndexMap
}

//The following functions are for interacting with the private data in
//IndexedFile as if it were public. The fields are private so they don't get
//marshalled into MongoDB

//SetHeader sets the bro header on the indexed file
func (i *IndexedFile) SetHeader(header *BroHeader) {
	i.header = header
}

//GetHeader retrieves the bro header on the indexed file
func (i *IndexedFile) GetHeader() *BroHeader {
	return i.header
}

//SetBroDataFactory sets the function which makes bro data corresponding
//with this type of bro file
func (i *IndexedFile) SetBroDataFactory(broDataFactory func() pt.BroData) {
	i.broDataFactory = broDataFactory
}

//GetBroDataFactory retrieves the function which makes bro data corresponding
//with this type of bro file
func (i *IndexedFile) GetBroDataFactory() func() pt.BroData {
	return i.broDataFactory
}

//SetFieldMap sets the map which maps the names of bro fields to the index
//in their respective bro data structs
func (i *IndexedFile) SetFieldMap(fieldMap BroHeaderIndexMap) {
	i.fieldMap = fieldMap
}

//GetFieldMap retrieves the map which maps the names of bro fields to the index
//in their respective bro data structs
func (i *IndexedFile) GetFieldMap() BroHeaderIndexMap {
	return i.fieldMap
}
