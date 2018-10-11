package commands

import (
	"time"
	"strings"
	"strconv"
	"context"

	"github.com/google/go-github/github"
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

	//Check Logs for Versioning
	m := res.MetaDB
	timestamp, version := m.LastCheck()

	days := time.Now().Sub( timestamp ).Hours()/24

	if days > float64(delta) {

		version, err := getRemoteVersion()

		if err == nil {
			return ""
		}

		//Log checked version.
		res.Log.WithFields(log.Fields{
			"Message":  "Checking versions...",
			"LastUpdateCheck": time.Now(),
			"NewestVersion": version,
		}).Info("Checking for new version")
	}

	if ( config.Version != version ) {
		return informUser( version )
	}

	return ""
}

// Versions look like "v#.#.#". Returns just the numbers.
func parseVersion( s string ) []string {
	t := strings.Trim(s, "v\n")
	return strings.Split( t, "." )
}

func getRemoteVersion() (string, error){
	client := github.NewClient(nil)
	refs, _, err := client.Git.GetRefs( context.Background(), "activecm", "rita", "refs/tags/v")

	if err == nil {
		s := strings.TrimPrefix( *refs[len(refs)-1].Ref, "refs/tags/")
		return s, nil
	} else {
		return "", err
	}
}

// Assembles a notice for the user informing them of an upgrade.
// The return value is printed regardless so, "" is returned on errror.
func informUser( verStr string ) string {
	var ret string;
	var versions [3]string;
	versions[0] = "Major"
	versions[1] = "Minor"
	versions[2] = "Patch"

	parsedRemoteVers := parseVersion( verStr )
	parsedLocalVers := parseVersion( config.Version )

	ret += "\nTheres a new "

	for i := 0; i < len(versions); i++ {
		remote, err1 := strconv.Atoi( parsedRemoteVers[i] )
		local, err2 := strconv.Atoi( parsedLocalVers[i] )

		if err1 != nil || err2 != nil {
			return ""
		}

		if  remote > local {
			ret += versions[i] + " "
			break
		}
	}
	ret += "version of RITA " + verStr + " available at:\n"
	ret += "https://github.com/activecm/rita\n"
	return ret
}
