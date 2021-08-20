// +build integration

package hostname

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo/dbtest"
)

// Server holds the dbtest DBServer
var Server dbtest.DBServer

// Set the test database
var testTargetDB = "tmp_test_db"

var testRepo Repository

func ipFactory(ip string) data.UniqueIP {
	return data.UniqueIP{
		IP:          ip,
		NetworkUUID: util.UnknownPrivateNetworkUUID,
		NetworkName: util.UnknownPrivateNetworkName,
	}
}

var testHostname = map[string]*Input{
	"a.b.activecountermeasures.com":   &Input{},
	"x.a.b.activecountermeasures.com": &Input{},
	"activecountermeasures.com":       &Input{},
	"google.com":                      &Input{},
}

func init() {
	testHostname["a.b.activecountermeasures.com"].ClientIPs.Insert(ipFactory("192.168.1.1"))
	testHostname["a.b.activecountermeasures.com"].ResolvedIPs.Insert(ipFactory("127.0.0.1"))
	testHostname["a.b.activecountermeasures.com"].ResolvedIPs.Insert(ipFactory("127.0.0.2"))

	testHostname["x.a.b.activecountermeasures.com"].ClientIPs.Insert(ipFactory("192.168.1.1"))
	testHostname["x.a.b.activecountermeasures.com"].ResolvedIPs.Insert(ipFactory("127.0.0.1"))
	testHostname["x.a.b.activecountermeasures.com"].ResolvedIPs.Insert(ipFactory("127.0.0.2"))

	testHostname["activecountermeasures.com"].ClientIPs.Insert(ipFactory("192.168.1.1"))

	testHostname["google.com"].ClientIPs.Insert(ipFactory("192.168.1.1"))
	testHostname["google.com"].ClientIPs.Insert(ipFactory("192.168.1.2"))

	testHostname["google.com"].ResolvedIPs.Insert(ipFactory("127.0.0.1"))
	testHostname["google.com"].ResolvedIPs.Insert(ipFactory("127.0.0.2"))
	testHostname["google.com"].ResolvedIPs.Insert(ipFactory("0.0.0.0"))

}

func TestUpsert(t *testing.T) {
	testRepo.Upsert(testHostname)
}

// TestMain wraps all tests with the needed initialized mock DB and fixtures
func TestMain(m *testing.M) {
	// Store temporary databases files in a temporary directory
	tempDir, _ := ioutil.TempDir("", "testing")
	Server.SetPath(tempDir)

	// Set the main session variable to the temporary MongoDB instance
	res := resources.InitTestResources()

	testRepo = NewMongoRepository(res.DB, res.Config, res.Log)

	// Run the test suite
	retCode := m.Run()

	// Shut down the temporary server and removes data on disk.
	Server.Stop()

	// call with result of m.Run()
	os.Exit(retCode)
}
