package commands

import (
	"time"
	"strings"
	"context"
	"fmt"

	"github.com/blang/semver"
	"github.com/urfave/cli"
	"github.com/google/go-github/github"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/config"
	log "github.com/sirupsen/logrus"
)

//Strings used for informing the user of a new version.
var informFmtStr string = "\nTheres a new %s version of RITA %s available at:\nhttps://github.com/activecm/rita/releases\n"
var versions = []string{ "Major", "Minor", "Patch" }

func GetVersionPrinter() func( *cli.Context){
	return  func(c *cli.Context) {
		fmt.Printf("%s version %s\n", c.App.Name, c.App.Version)
		fmt.Printf( updateCheck(c.String("config")) )
	}
}

// UpdateCheck Performs a check for the new version of RITA against the git repository and
//returns a string indicating the new version if available
func updateCheck( configFile string) string {
	res := resources.InitResources( configFile )
	deltaPtr := res.Config.S.UserConfig.UpdateCheckFrequency
	var newVersion semver.Version
	var err error
	var delta int
	var timestamp time.Time

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
	timestamp, newVersion = m.LastCheck()

	days := time.Now().Sub( timestamp ).Hours()/24

	if days > float64(delta) {
		newVersion, err = getRemoteVersion()

		if err != nil {
			return ""
		}

		//Log checked version.
		res.Log.WithFields(log.Fields{
			"Message":  "Checking versions...",
			"LastUpdateCheck": time.Now(),
			"NewestVersion": fmt.Sprint(newVersion),
		}).Info("Checking for new version")

	}

	configVersion ,err := semver.ParseTolerant( config.Version )
	if err != nil {
		return ""
	}

	if newVersion.GT(configVersion) {
		return informUser( configVersion, newVersion )
	}

	return ""
}

// Returns the first index where v1 is greater than v2
func versionDiffIndex( v1 semver.Version, v2 semver.Version) int {

	if v1.Major > v2.Major {
		return 0
	}
	if v1.Minor > v2.Minor {
		return 1
	}

	return 2
}

func getRemoteVersion() (semver.Version, error){
	client := github.NewClient(nil)
	refs, _, err := client.Git.GetRefs( context.Background(), "activecm", "rita", "refs/tags/v")

	if err == nil {
		s := strings.TrimPrefix( *refs[len(refs)-1].Ref, "refs/tags/")
		return semver.ParseTolerant(s)
	} else {
		return semver.Version{}, err
	}
}

// Assembles a notice for the user informing them of an upgrade.
// The return value is printed regardless so, "" is returned on errror.
//func informUser( verStr string, index int ) string {
func informUser( local semver.Version, remote semver.Version ) string {

	return fmt.Sprintf(informFmtStr,
			versions[versionDiffIndex(remote,local)],
			fmt.Sprint(remote))
}
