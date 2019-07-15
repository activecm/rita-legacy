package commands

import (
	"testing"
	"fmt"

	"github.com/stretchr/testify/assert"

	"github.com/activecm/rita/config"
)

func TestSetRolling(t *testing.T) {
	type cfg = config.RollingStaticCfg // including the definition here for reference:
	// 	DefaultChunks int `yaml:"DefaultChunks" default:"12"`
	// 	Rolling       bool
	// 	CurrentChunk  int
	// 	TotalChunks   int

	type tc struct{
		msg string
		dbExists bool
		dbIsRolling bool
		dbCurrChunk int
		dbTotalChunks int
		userIsRolling bool
		userCurrChunk int
		userTotalChunks int
		cfgDefaultChunks int
		expected cfg
		err bool
	}

	// this is the sentinel value that signifies that a user did not supply
	// a command line value for --chunk or --numchunks
	const blank int = -1
	// these are used to help make the test table below (a little) more readable
	const exists bool = true
	const rolling bool = true
	const returnsError bool = true
	const default12 int = 12
	const default24 int = 24

	testCases := []tc{
		// new database scenarios

		tc{"rita import (default 12)",
			!exists, !rolling, 0, 0, !rolling, blank, blank, default12, cfg{0, !rolling, 0, 1}, !returnsError},

		tc{"rita import --rolling (default 12)",
			!exists, !rolling, 0, 0, rolling, blank, blank, default12, cfg{0, rolling, 0, 12}, !returnsError},

		tc{"rita import --rolling --chunk 0 --numchunks 24 (default 12)",
			!exists, !rolling, 0, 0, rolling, 0, 24, default12, cfg{0, rolling, 0, 24}, !returnsError},

		tc{"rita import --numchunks 24 (default 12)",
			!exists, !rolling, 0, 0, !rolling, blank, 24, default12, cfg{0, rolling, 0, 24}, !returnsError},

		tc{"rita import --chunk 5  (default 12)",
			!exists, !rolling, 0, 0, !rolling, 5, blank, default12, cfg{0, rolling, 5, 12}, !returnsError},

		tc{"rita import --chunk 12 (default 12)",
			!exists, !rolling, 0, 0, !rolling, 12, blank, default12, cfg{0, rolling, 12, 12}, returnsError},

		tc{"rita import --chunk 12 (default 24)",
			!exists, !rolling, 0, 0, !rolling, 12, blank, default24, cfg{0, rolling, 12, 24}, !returnsError},

		tc{"rita import --chunk 12 --numchunks 24 (default 12)",
			!exists, !rolling, 0, 0, !rolling, 12, 24, default12, cfg{0, rolling, 12, 24}, !returnsError},

		tc{"rita import --chunk -2 (default 12)",
			!exists, !rolling, 0, 0, !rolling, -2, blank, default12, cfg{0, rolling, -2, 12}, returnsError},

		tc{"rita import --numchunks -2 (default 12)",
			!exists, !rolling, 0, 0, !rolling, blank, -2, default12, cfg{0, rolling, 0, -2}, returnsError},

		// existing database scenarios

		// non-rolling, current chunk 0, total chunks 1
		tc{"rita import",
			exists, !rolling, 0, 1, !rolling, blank, blank, default12, cfg{0, rolling, 1, 12}, !returnsError},

		tc{"rita import --rolling",
			exists, !rolling, 0, 1, rolling, blank, blank, default12, cfg{0, rolling, 1, 12}, !returnsError},

		tc{"rita import --rolling --chunk 0 --numchunks 24",
			exists, !rolling, 0, 1, rolling, 0, 24, default12, cfg{0, rolling, 0, 24}, !returnsError},

		tc{"rita import --numchunks 24",
			exists, !rolling, 0, 1, !rolling, blank, 24, default12, cfg{0, rolling, 1, 24}, !returnsError},

		tc{"rita import --chunk 5 (default 12)",
			exists, !rolling, 0, 1, !rolling, 5, blank, default12, cfg{0, rolling, 5, 12}, !returnsError},

		tc{"rita import --chunk 12 (default 12)",
			exists, !rolling, 0, 1, !rolling, 12, blank, default12, cfg{0, rolling, 12, 12}, returnsError},

		tc{"rita import --chunk 12 (default 24)",
			exists, !rolling, 0, 1, !rolling, 12, blank, default24, cfg{0, rolling, 12, 24}, !returnsError},

		tc{"rita import --chunk 12 --numchunks 24",
			exists, !rolling, 0, 1, !rolling, 12, 24, default12, cfg{0, rolling, 12, 24}, !returnsError},

		tc{"rita import --chunk -2",
			exists, !rolling, 0, 1, !rolling, -2, blank, default12, cfg{0, rolling, -2, 12}, returnsError},

		tc{"rita import --numchunks -2",
			exists, !rolling, 0, 1, !rolling, blank, -2, default12, cfg{0, rolling, 1, -2}, returnsError},

		// rolling, current chunk 1, total chunks 12
		tc{"rita import",
			exists, rolling, 1, 12, !rolling, blank, blank, default12, cfg{0, rolling, 2, 12}, !returnsError},

		tc{"rita import --rolling",
			exists, rolling, 1, 12, rolling, blank, blank, default12, cfg{0, rolling, 2, 12}, !returnsError},

		tc{"rita import --rolling --chunk 0 --numchunks 24",
			exists, rolling, 1, 12, rolling, 0, 24, default12, cfg{0, rolling, 0, 24}, returnsError},

		tc{"rita import --numchunks 24",
			exists, rolling, 1, 12, !rolling, blank, 24, default12, cfg{0, rolling, 2, 24}, returnsError},

		tc{"rita import --chunk 5 (default 12)",
			exists, rolling, 1, 12, !rolling, 5, blank, default12, cfg{0, rolling, 5, 12}, !returnsError},

		tc{"rita import --chunk 12 (default 12)",
			exists, rolling, 1, 12, !rolling, 12, blank, default12, cfg{0, rolling, 12, 12}, returnsError},

		tc{"rita import --chunk 12 (default 24)",
			exists, rolling, 1, 12, !rolling, 12, blank, default24, cfg{0, rolling, 12, 12}, returnsError},

		tc{"rita import --chunk 12 --numchunks 24",
			exists, rolling, 1, 12, !rolling, 12, 24, default12, cfg{0, rolling, 12, 24}, returnsError},

		tc{"rita import --chunk -2",
			exists, rolling, 1, 12, !rolling, -2, blank, default12, cfg{0, rolling, -2, 12}, returnsError},

		tc{"rita import --numchunks -2",
			exists, rolling, 1, 12, !rolling, blank, -2, default12, cfg{0, rolling, 0, -2}, returnsError},

		// rolling, current chunk 11, total chunks 12
		tc{"rita import",
			exists, rolling, 11, 12, !rolling, blank, blank, default12, cfg{0, rolling, 0, 12}, !returnsError},

		tc{"rita import --rolling",
			exists, rolling, 11, 12, rolling, blank, blank, default12, cfg{0, rolling, 0, 12}, !returnsError},

		tc{"rita import --rolling --chunk 0 --numchunks 24",
			exists, rolling, 11, 12, rolling, 0, 24, default12, cfg{0, rolling, 0, 24}, returnsError},

		tc{"rita import --numchunks 24",
			exists, rolling, 11, 12, !rolling, blank, 24, default12, cfg{0, rolling, 12, 24}, returnsError},

		tc{"rita import --chunk 5 (default 12)",
			exists, rolling, 11, 12, !rolling, 5, blank, default12, cfg{0, rolling, 5, 12}, !returnsError},

		tc{"rita import --chunk 12 (default 12)",
			exists, rolling, 11, 12, !rolling, 12, blank, default12, cfg{0, rolling, 12, 12}, returnsError},

		tc{"rita import --chunk 12 (default 24)",
			exists, rolling, 11, 12, !rolling, 12, blank, default24, cfg{0, rolling, 12, 12}, returnsError},

		tc{"rita import --chunk 12 --numchunks 24",
			exists, rolling, 11, 12, !rolling, 12, 24, default12, cfg{0, rolling, 12, 24}, returnsError},

		// rolling, current chunk 11, total chunks 24
		tc{"rita import",
			exists, rolling, 11, 24, !rolling, blank, blank, default12, cfg{0, rolling, 12, 24}, !returnsError},

		tc{"rita import --rolling",
			exists, rolling, 11, 24, rolling, blank, blank, default12, cfg{0, rolling, 12, 24}, !returnsError},

		tc{"rita import --rolling --chunk 0 --numchunks 24",
			exists, rolling, 11, 24, rolling, 0, 24, default12, cfg{0, rolling, 0, 24}, !returnsError},

		tc{"rita import --numchunks 12",
			exists, rolling, 11, 24, !rolling, blank, 12, default12, cfg{0, rolling, 0, 12}, returnsError},

		tc{"rita import --chunk 12 --numchunks 12",
			exists, rolling, 11, 24, !rolling, 12, 12, default12, cfg{0, rolling, 12, 12}, returnsError},

		tc{"rita import --chunk 13 (default 12)",
			exists, rolling, 11, 24, !rolling, 13, blank, default12, cfg{0, rolling, 13, 24}, !returnsError},
	}

	// runner for the test table above
	for _, testCase := range testCases {
		actual, err := setRolling(
			testCase.dbExists, testCase.dbIsRolling, testCase.dbCurrChunk, testCase.dbTotalChunks,
			testCase.userIsRolling,	testCase.userCurrChunk,	testCase.userTotalChunks, testCase.cfgDefaultChunks,
		)

		// Construct a message that will help pinpoint which test failed
		var dbMsg string
		if testCase.dbExists && testCase.dbIsRolling {
			dbMsg = fmt.Sprintf("rolling, currChunk=%d, totalChunks=%d", testCase.dbCurrChunk, testCase.dbTotalChunks)
		} else if testCase.dbExists && !testCase.dbIsRolling {
			dbMsg = fmt.Sprintf("non-rolling, currChunk=%d, totalChunks=%d", testCase.dbCurrChunk, testCase.dbTotalChunks)
		} else {
			dbMsg = "new"
		}

		if testCase.err {
			assert.Errorf(t, err, "db: <%s> cmd: <%s>", dbMsg, testCase.msg)
		} else {
			assert.Equalf(t, testCase.expected, actual, "db: <%s> cmd: <%s>", dbMsg, testCase.msg)
		}
	}

}
