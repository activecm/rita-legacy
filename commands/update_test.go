package commands

import (
	"fmt"
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
)

func TestVersionCompare(t *testing.T) {

	Base, _ := semver.Parse("1.1.1")
	Major, _ := semver.Parse("2.3.4")
	Minor, _ := semver.Parse("1.2.3")
	Patch, _ := semver.Parse("1.1.9")

	assert.Equal(t, 0, versionDiffIndex(Major, Base),
		"Should return new Major")
	assert.Equal(t, 1, versionDiffIndex(Minor, Base),
		"Should return new Minor")
	assert.Equal(t, 2, versionDiffIndex(Patch, Base),
		"Should return new Patch")

}

func TestInformUser(t *testing.T) {

	Base, _ := semver.Parse("1.1.1")
	Major, _ := semver.Parse("2.3.4")
	assert.Equal(t,
		fmt.Sprintf(informFmtStr, "Major", "2.3.4"),
		informUser(Base, Major),
		"Should be identical strings")
}
