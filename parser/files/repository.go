package files

import (
	"time"

	pt "github.com/activecm/rita/parser/parsetypes"
	"github.com/globalsign/mgo/bson"
)

//BroHeader contains the parse information contained within the comment lines
//of Zeek files
type BroHeader struct {
	Names     []string // Names of fields
	Types     []string // Types of fields
	Separator string   // Field separator
	SetSep    string   // Set separator
	Empty     string   // Empty field tag
	Unset     string   // Unset field tag
	ObjType   string   // Object type (comes from #path)
}

//ZeekHeaderIndexMap maps the indexes of the fields in the ZeekHeader to the respective
//indexes in the parsetype.BroData structs
type ZeekHeaderIndexMap struct {
	NthLogFieldExistsInParseType []bool
	NthLogFieldParseTypeOffset   []int
}

//IndexedFile ties a file to a target collection and database
type IndexedFile struct {
	ID               bson.ObjectId `bson:"_id,omitempty"`
	Path             string        `bson:"filepath"`
	Length           int64         `bson:"length"`
	ModTime          time.Time     `bson:"modified"`
	Hash             string        `bson:"hash"`
	TargetCollection string        `bson:"collection"`
	TargetDatabase   string        `bson:"database"`
	CID              int           `bson:"cid"`
	ParseTime        time.Time     `bson:"time_complete"`
	header           *BroHeader
	broDataFactory   func() pt.BroData
	fieldMap         ZeekHeaderIndexMap
	json             bool
}

//The following functions are for interacting with the private data in
//IndexedFile as if it were public. The fields are private so they don't get
//marshalled into MongoDB

//IsJSON returns whether the file is a json file
func (i *IndexedFile) IsJSON() bool {
	return i.json
}

//SetJSON sets the json flag
func (i *IndexedFile) SetJSON() {
	i.json = true
}

//SetHeader sets the broHeader on the indexed file
func (i *IndexedFile) SetHeader(header *BroHeader) {
	i.header = header
}

//GetHeader retrieves the broHeader on the indexed file
func (i *IndexedFile) GetHeader() *BroHeader {
	return i.header
}

//SetBroDataFactory sets the function which makes broData corresponding
//with this type of Zeek file
func (i *IndexedFile) SetBroDataFactory(broDataFactory func() pt.BroData) {
	i.broDataFactory = broDataFactory
}

//GetBroDataFactory retrieves the function which makes broData corresponding
//with this type of Zeek file
func (i *IndexedFile) GetBroDataFactory() func() pt.BroData {
	return i.broDataFactory
}

//SetFieldMap sets the map which maps the indexes of Zeek fields in the log header to the indexes
//in their respective broData structs
func (i *IndexedFile) SetFieldMap(fieldMap ZeekHeaderIndexMap) {
	i.fieldMap = fieldMap
}

//GetFieldMap retrieves the map which maps the indexes of Zeek fields in the log header to the indexes
//in their respective broData structs
func (i *IndexedFile) GetFieldMap() ZeekHeaderIndexMap {
	return i.fieldMap
}
