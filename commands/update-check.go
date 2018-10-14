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
	deltaPtr := res.Config.S.UserConfig.UpdateCheckFrequency
	var version string
	var delta int

	if deltaPtr == nil {
		delta = 14
	} else {
		delta = *deltaPtr
	}

	if delta <= 0 {
		return ""
	}

	//Check Logs for Versioning
	m := res.MetaDB
	timestamp, version := m.LastCheck()

	days := time.Now().Sub( timestamp ).Hours()/24

	if days > float64(delta) {
		version, err := getRemoteVersion()

		if err != nil {
			return ""
		}

		//Log checked version.
		res.Log.WithFields(log.Fields{
			"Message":  "Checking versions...",
			"LastUpdateCheck": time.Now(),
			"NewestVersion": version,
		}).Info("Checking for new version")
	}

	index, err := versionCmp( version, config.Version )
	if err == nil  && index != -1 {
		return informUser( version, index )
	}

	return ""
}

// Checks if s1 is newer than s2
// Returns the first index where s1 is greater than s2
// Returns -1 if s2 is newer than s1
func versionCmp( s1 string, s2 string) (int, error) {
	arr1 := parseVersion( s1 )
	arr2 := parseVersion( s2 )
	var i int

	for i = 0; i < len(arr1) || i < len(arr2); i++ {

		val1, err1 := strconv.Atoi( arr1[i] )
		if err1 != nil {
			return -1, err1
		}

		val2, err2 := strconv.Atoi( arr2[i] )
		if err2 != nil {
			return -1, err2
		}

		if  val1 > val2 {
			return i, nil
		}
	}

	return -1, nil
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
func informUser( verStr string, index int ) string {
	var ret string;
	var versions [3]string;
	versions[0] = "Major "
	versions[1] = "Minor "
	versions[2] = "Patch "

	ret += "\nTheres a new " + versions[index]
	ret += "version of RITA " + verStr + " available at:\n"
	ret += "https://github.com/activecm/rita\n"
	return ret
}
