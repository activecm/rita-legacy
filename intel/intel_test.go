package intel

import (
	"github.com/ocmdev/rita/config"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/Sirupsen/logrus/hooks/test"
	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2"
)

// Hooks up go's base testing runner to check
func Test(t *testing.T) { TestingT(t) }

type IntelSuite struct {
	conf       *config.Resources
	tob        *IntelHandle
	loghook    *test.Hook
	cymruTests []string
}

var _ = Suite(&IntelSuite{})

// SetUpTest will stand up the testing structure
func (i *IntelSuite) SetUpTest(c *C) {
	i.conf = config.InitConfigFromAlternatePath("../testing_resources/testconfig.yaml")
	i.cymruTests = []string{
		"20940   | 23.15.8.88       | 23.15.8.0/23        | US | arin     | 2010-12-17 | AKAMAI-ASN1 , US",
		"20940   | 23.15.8.88       | 23.15.8.0/23        | US | arin     | 2010-12-17 ",
		"20940   | 23.15.8.88       | 23.15.8.0/23        | US | arin     | 20100-12-17 | AKAMAI-ASN1 , US",
		"23028   | 216.90.108.31    | 216.90.108.0/24     | US | arin     |            | TEAM-CYMRU - Team Cymru Inc., US",
		"NA   | 23.15.8.88       | 23.15.8.0/23        | US | arin     | 2010-12-17 ",
	}

	tob := NewIntelHandle(i.conf)
	c.Assert(tob.intelDB, DeepEquals, i.conf.System.HostIntelDB)
	c.Assert(tob.session, FitsTypeOf, new(mgo.Session))
	c.Assert(tob.log, FitsTypeOf, new(log.Logger))
	c.Assert(tob.baseInstallDir, Equals, i.conf.System.BaseInstallDirectory)
	i.tob = tob
	i.tob.log, i.loghook = test.NewNullLogger()

}

// TestparseCymruLine runs a series of tests against the line parser that check for
// correctness of parsing as well as failures of validation, and erronious lines.
func (i *IntelSuite) TestparseCymruLine(c *C) {

	// Test with a line as we expect to see them
	testOk, err := i.tob.parseCymruLine(i.cymruTests[0])
	c.Assert(err, IsNil)
	c.Assert(testOk.ASN, DeepEquals, int64(20940))
	c.Assert(testOk.IP, DeepEquals, string("23.15.8.88"))
	c.Assert(testOk.Prefix, DeepEquals, string("23.15.8.0/23"))
	c.Assert(testOk.CountryCode, DeepEquals, string("US"))
	c.Assert(testOk.Registry, DeepEquals, string("arin"))
	checkTime, _ := time.Parse("2006-01-02", "2010-12-17")
	c.Assert(testOk.Allocated, DeepEquals, checkTime)
	c.Assert(testOk.ASName, DeepEquals, string("AKAMAI-ASN1 , US"))

	// Test with a line that has too few fields
	_, err = i.tob.parseCymruLine(i.cymruTests[1])
	c.Assert(err, Equals, PEBadCymruFieldCount)

	// Test a line with a bad date
	testOk2, err := i.tob.parseCymruLine(i.cymruTests[2])
	talloc, _ := time.Parse("2006-01-02", "1970-01-01")
	c.Assert(testOk2.Allocated, Equals, talloc)

	// Test a missing field in the middle
	testOk3, err := i.tob.parseCymruLine(i.cymruTests[3])
	c.Assert(err, IsNil)
	c.Assert(testOk3.ASN, Equals, int64(23028))
	i.loghook.Reset()

}

// TestValidCymruInput runs tests against the input validator
func (i *IntelSuite) TestValidCymruInput(c *C) {
	c.Assert(i.tob.validCymruInput("192.168.1.0"), Equals, false)
	c.Assert(len(i.loghook.Entries), Equals, 1)
	c.Assert(i.tob.validCymruInput("aabc.def"), Equals, false)
	c.Assert(2, Equals, len(i.loghook.Entries))
	i.loghook.Reset()
}

// TestCymruWhoisLookup runs a series of tests against the whois lookup code.
// Note that these test are dependant on a network connection that can reach
// the team cymru servers.
func (i *IntelSuite) TestCymruWhoisLookup(c *C) {
	// Zero lenght string should log an error
	result := i.tob.CymruWhoisLookup([]string{})
	c.Assert(1, Equals, len(i.loghook.Entries))
	i.loghook.Reset()

	// Send in an address and check that we get the correct result
	result = i.tob.CymruWhoisLookup([]string{"216.90.108.31"})

	//c.Assert(len(i.loghook.Entries), Equals, 1)
	// we check to see if we get the same ASN
	testout, err := i.tob.parseCymruLine(i.cymruTests[3])
	c.Assert(err, IsNil)
	c.Assert(len(result), Equals, 1)
	c.Assert(result[0].ASN, Equals, testout.ASN)

	i.loghook.Reset()
}
