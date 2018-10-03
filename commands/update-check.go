package commands

import (
	"time"
	"net/http"
	"io/ioutil"
	"strings"
	"strconv"

	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/config"
	log "github.com/sirupsen/logrus"
)

// UpdateCheck Performs a check for the new version of RITA against the git repository and
//returns a string indicating the new version if available
func UpdateCheck( configFile string) string {
	res := resources.InitResources( configFile )
	delta := res.Config.S.UserConfig.UpdateCheckFrequency

	if delta == 0 {
		return ""
	}

	//Check Log for Versioning
	m := res.MetaDB
	t, s := m.LastCheck()

	days := time.Now().Sub( t ).Hours()/24

	if days > float64(delta) {

		s = remoteVersion()

		if s == "" {
			return ""
		}

		res.Log.WithFields(log.Fields{
			"Message":  "Checking versions...",
			"LastUpdateCheck": time.Now(),
			"NewestVersion": s,
		}).Info("Checking for new version")
	}

	if ( config.Version != s ) {
		return informUser( s )
	}

	return ""
}

func parseVersion( s string ) []string {
	t := strings.Trim(s, "v\n")
	return strings.Split( t, "." )
}

func remoteVersion() string {
	resp, err := http.Get("https://raw.githubusercontent.com/zaowen/rita/UpdateCheck/Version")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	version, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return strings.Trim(string(version), "\n")
}

func informUser( s string ) string {
	var ret string;
	var versions [3]string;
	versions[0] = "Major"
	versions[1] = "Minor"
	versions[2] = "Patch"

	ps := parseVersion( s )
	pv := parseVersion( config.Version )

	ret += "\nTheres a new "

	for i := 0; i < len(versions); i++ {
		psi, _ := strconv.Atoi( ps[i] )
		pvi, _ := strconv.Atoi( pv[i] )
		if psi > pvi {
			ret += versions[i] + " "
			break
		}
	}
	ret += "version of RITA " + s + " available at:\n"
	ret += "https://github.com/activecm/rita\n"
	return ret
}
