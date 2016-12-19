package inteldb

import (
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/datatypes/intel"
	"testing"
	"time"

	"github.com/Sirupsen/logrus/hooks/test"
	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2"
)

// Hooks up go's base testing runner to check
func Test(t *testing.T) { TestingT(t) }

// IntelDBTestSuite defines a testing suite for the inteldb toolkit
type IntelDBTestSuite struct {
	conf    *config.Resources
	tob     *IntelDBHandle
	loghook *test.Hook
}

var _ = Suite(&IntelDBTestSuite{})

// SetUpTest will stand up the testing structures
func (i *IntelDBTestSuite) SetUpSuite(c *C) {
	i.conf = config.InitConfigFromAlternatePath("../../testing_resources/testconfig.yaml")
	// drop the test database if one exists so that we can check that our
	// constructor builds one if it's not present
	tssn := i.conf.CopySession()
	defer tssn.Close()
	names, _ := tssn.DatabaseNames()
	for _, name := range names {
		if name == i.conf.System.HostIntelDB {
			err := tssn.DB(i.conf.System.HostIntelDB).DropDatabase()
			if err != nil {
				panic(err)
			}
		}
	}
}

// TestConstructedObject does sanity checks on the returned object
func (i *IntelDBTestSuite) TestConstructedObject(c *C) {
	// Test that a constructed object has the side affect of creating
	// the database if there isn't one already there
	i.tob = NewIntelDBHandle(i.conf)
	tssn := i.conf.CopySession()
	defer tssn.Close()
	names, _ := tssn.DatabaseNames()
	found := false
	for _, name := range names {
		if name == i.conf.System.HostIntelDB {
			found = true
		}
	}
	if !found {
		c.FailNow()
	}

	// Check that each of the types is correct in our new object
	c.Assert(i.tob.conf, FitsTypeOf, new(config.Resources))
	c.Assert(i.tob.db, Equals, i.conf.System.HostIntelDB)
	c.Assert(i.tob.ssn, FitsTypeOf, new(mgo.Session))

	// Set the logger to a testing logger
	i.tob.log, i.loghook = test.NewNullLogger()

}

// TestReadWrite does some simple reading and writing to see if the read and write
// features are working with the database.
func (i *IntelDBTestSuite) TestReadWrite(c *C) {
	// Start by resetting loghook so we don't get panics
	i.loghook.Reset()

	// create a data point to be written out to the database
	tdata := data.IntelData{
		HostName:    "testdata.com",
		IP:          "8.9.10.11",
		ASN:         23345,
		Prefix:      "8.9.10.0/24",
		CountryCode: "US",
		Registry:    "testreg",
		Allocated:   time.Now(),
		Info:        "none",
		ASName:      "Testing Inc.",
		IntelDate:   time.Now(),
	}

	// attempt to create a record in the database
	i.tob.Write(tdata)

	// sleep so the database has a moment to process
	time.Sleep(1 * time.Second)

	// attempt to read the data back out and check that it works
	var bdata data.IntelData
	err := i.tob.Find("8.9.10.11").IntelData(&bdata)
	c.Assert(err, IsNil)
	// really simple check to see if we got the right data
	c.Assert(tdata.ASN, Equals, bdata.ASN)

}

// TestSetAndGetBlacklistedScore tests the SetBlacklistedScore function
func (i *IntelDBTestSuite) TestSetAndGetBlacklistedScore(c *C) {

	// Set the score for the object in the previous test
	err := i.tob.Find("8.9.10.11").SetBlacklistedScore(9)
	c.Assert(err, IsNil)
	// Check to see if we can retrieve that score
	score, err := i.tob.Find("8.9.10.11").GetBlacklistedScore()
	c.Assert(err, IsNil)
	c.Assert(score, DeepEquals, int(9))
}

// TearDownSuite allows the test object to close out its write loop
func (i *IntelDBTestSuite) TearDownSuite(c *C) {
	i.tob.Close()
}
