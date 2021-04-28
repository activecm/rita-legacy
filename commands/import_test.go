package commands

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/activecm/rita/config"
)

func TestParseFlags(t *testing.T) {
	type cfg = config.RollingStaticCfg // including the definition here for reference:
	// 	DefaultChunks int `yaml:"DefaultChunks" default:"12"`
	// 	Rolling       bool
	// 	CurrentChunk  int
	// 	TotalChunks   int

	type tc struct {
		msg              string
		dbExists         bool
		dbIsRolling      bool
		dbCurrChunk      int
		dbTotalChunks    int
		userIsRolling    bool
		userCurrChunk    int
		userTotalChunks  int
		cfgDefaultChunks int
		deleteOldData    bool
		expected         cfg
		err              bool
	}

	// this is the sentinel value that signifies that a user did not supply
	// a command line value for --chunk or --numchunks
	const blank int = -1
	// these are used to help make the test table below (a little) more readable
	const exists bool = true
	const rolling bool = true
	const delete bool = true
	const returnsError bool = true
	const default12 int = 12
	const default24 int = 24

	testCases := []tc{
		// new database scenarios

		{"rita import (default 12)",
			!exists, !rolling, 0, 0, !rolling, blank, blank, default12, !delete, cfg{12, !rolling, 0, 1}, !returnsError},

		{"rita import --rolling (default 12)",
			!exists, !rolling, 0, 0, rolling, blank, blank, default12, !delete, cfg{12, rolling, 0, 12}, !returnsError},

		{"rita import --rolling --chunk 0 --numchunks 24 (default 12)",
			!exists, !rolling, 0, 0, rolling, 0, 24, default12, !delete, cfg{12, rolling, 0, 24}, !returnsError},

		{"rita import --numchunks 24 (default 12)",
			!exists, !rolling, 0, 0, !rolling, blank, 24, default12, !delete, cfg{12, rolling, 0, 24}, !returnsError},

		{"rita import --chunk 5  (default 12)",
			!exists, !rolling, 0, 0, !rolling, 5, blank, default12, !delete, cfg{12, rolling, 5, 12}, !returnsError},

		{"rita import --chunk 12 (default 12)",
			!exists, !rolling, 0, 0, !rolling, 12, blank, default12, !delete, cfg{12, rolling, 12, 12}, returnsError},

		{"rita import --chunk 12 (default 24)",
			!exists, !rolling, 0, 0, !rolling, 12, blank, default24, !delete, cfg{24, rolling, 12, 24}, !returnsError},

		{"rita import --chunk 12 --numchunks 24 (default 12)",
			!exists, !rolling, 0, 0, !rolling, 12, 24, default12, !delete, cfg{12, rolling, 12, 24}, !returnsError},

		{"rita import --chunk -2 (default 12)", // error reason: chunk number must be positive
			!exists, !rolling, 0, 0, !rolling, -2, blank, default12, !delete, cfg{}, returnsError},

		{"rita import --numchunks -2 (default 12)", // error reason: numchunks must be positive
			!exists, !rolling, 0, 0, !rolling, blank, -2, default12, !delete, cfg{}, returnsError},

		{"rita import --delete (default 12)",
			!exists, !rolling, 0, 0, !rolling, blank, blank, default12, delete, cfg{12, !rolling, 0, 1}, !returnsError},

		{"rita import --delete --rolling (default 12)",
			!exists, !rolling, 0, 0, rolling, blank, blank, default12, delete, cfg{12, rolling, 0, 12}, !returnsError},

		{"rita import --delete --rolling --chunk 0 --numchunks 24 (default 12)",
			!exists, !rolling, 0, 0, rolling, 0, 24, default12, delete, cfg{12, rolling, 0, 24}, !returnsError},

		{"rita import --delete --chunk 5  (default 12)",
			!exists, !rolling, 0, 0, !rolling, 5, blank, default12, delete, cfg{12, rolling, 5, 12}, !returnsError},

		// existing database scenarios

		// non-rolling, current chunk 0, total chunks 1
		{"rita import", // error reason: cannot import into existing non-rolling db
			exists, !rolling, 0, 1, !rolling, blank, blank, default12, !delete, cfg{}, returnsError},

		{"rita import --rolling",
			exists, !rolling, 0, 1, rolling, blank, blank, default12, !delete, cfg{12, rolling, 1, 12}, !returnsError},

		{"rita import --rolling --chunk 0 --numchunks 24",
			exists, !rolling, 0, 1, rolling, 0, 24, default12, !delete, cfg{12, rolling, 0, 24}, !returnsError},

		{"rita import --numchunks 24",
			exists, !rolling, 0, 1, !rolling, blank, 24, default12, !delete, cfg{12, rolling, 1, 24}, !returnsError},

		{"rita import --chunk 5 (default 12)",
			exists, !rolling, 0, 1, !rolling, 5, blank, default12, !delete, cfg{12, rolling, 5, 12}, !returnsError},

		{"rita import --chunk 12 (default 12)",
			exists, !rolling, 0, 1, !rolling, 12, blank, default12, !delete, cfg{12, rolling, 12, 12}, returnsError},

		{"rita import --chunk 12 (default 24)",
			exists, !rolling, 0, 1, !rolling, 12, blank, default24, !delete, cfg{24, rolling, 12, 24}, !returnsError},

		{"rita import --chunk 12 --numchunks 24",
			exists, !rolling, 0, 1, !rolling, 12, 24, default12, !delete, cfg{12, rolling, 12, 24}, !returnsError},

		{"rita import --chunk -2", // error reason: chunk number must be positive
			exists, !rolling, 0, 1, !rolling, -2, blank, default12, !delete, cfg{}, returnsError},

		{"rita import --numchunks -2", // error reason: numchunks must be positive
			exists, !rolling, 0, 1, !rolling, blank, -2, default12, !delete, cfg{}, returnsError},

		{"rita import --delete (default 12)",
			exists, !rolling, 0, 1, !rolling, blank, blank, default12, delete, cfg{12, !rolling, 0, 1}, !returnsError},

		{"rita import --delete --rolling (default 12)",
			exists, !rolling, 0, 1, rolling, blank, blank, default12, delete, cfg{12, rolling, 0, 12}, !returnsError},

		{"rita import --delete --chunk 5 (default 12)",
			exists, !rolling, 0, 1, !rolling, 5, blank, default12, delete, cfg{12, rolling, 5, 12}, !returnsError},

		{"rita import --delete --rolling --chunk 0 --numchunks 24 (default 12)",
			exists, !rolling, 0, 1, rolling, 0, 24, default12, delete, cfg{12, rolling, 0, 24}, !returnsError},

		// rolling, current chunk 1, total chunks 12
		{"rita import",
			exists, rolling, 1, 12, !rolling, blank, blank, default12, !delete, cfg{12, rolling, 2, 12}, !returnsError},

		{"rita import --rolling",
			exists, rolling, 1, 12, rolling, blank, blank, default12, !delete, cfg{12, rolling, 2, 12}, !returnsError},

		{"rita import --rolling --chunk 0 --numchunks 24",
			exists, rolling, 1, 12, rolling, 0, 24, default12, !delete, cfg{12, rolling, 0, 24}, !returnsError},

		{"rita import --numchunks 24",
			exists, rolling, 1, 12, !rolling, blank, 24, default12, !delete, cfg{12, rolling, 2, 24}, !returnsError},

		{"rita import --chunk 5 (default 12)",
			exists, rolling, 1, 12, !rolling, 5, blank, default12, !delete, cfg{12, rolling, 5, 12}, !returnsError},

		{"rita import --chunk 12 (default 12)", // error reason: chunk must be less than db numchunks
			exists, rolling, 1, 12, !rolling, 12, blank, default12, !delete, cfg{}, returnsError},

		{"rita import --chunk 12 (default 24)", // error reason: chunk must be less than db numchunks
			exists, rolling, 1, 12, !rolling, 12, blank, default24, !delete, cfg{}, returnsError},

		{"rita import --chunk 12 --numchunks 24",
			exists, rolling, 1, 12, !rolling, 12, 24, default12, !delete, cfg{12, rolling, 12, 24}, !returnsError},

		{"rita import --chunk -2", // error reason: chunk number must be positive
			exists, rolling, 1, 12, !rolling, -2, blank, default12, !delete, cfg{}, returnsError},

		{"rita import --numchunks -2", // error reason: numchunks must be positive
			exists, rolling, 1, 12, !rolling, blank, -2, default12, !delete, cfg{}, returnsError},

		{"rita import --delete (default 12)",
			exists, rolling, 1, 12, !rolling, blank, blank, default12, delete, cfg{12, rolling, 1, 12}, !returnsError},

		{"rita import --delete --rolling (default 12)",
			exists, rolling, 1, 12, !rolling, blank, blank, default12, delete, cfg{12, rolling, 1, 12}, !returnsError},

		{"rita import --delete --chunk 5 (default 12)",
			exists, rolling, 1, 12, !rolling, 5, blank, default12, delete, cfg{12, rolling, 5, 12}, !returnsError},

		{"rita import --delete --rolling --chunk 0 --numchunks 24 (default 12)",
			exists, rolling, 1, 12, rolling, 0, 24, default12, delete, cfg{12, rolling, 0, 24}, !returnsError},

		// rolling, current chunk 11, total chunks 12
		{"rita import",
			exists, rolling, 11, 12, !rolling, blank, blank, default12, !delete, cfg{12, rolling, 0, 12}, !returnsError},

		{"rita import --rolling",
			exists, rolling, 11, 12, rolling, blank, blank, default12, !delete, cfg{12, rolling, 0, 12}, !returnsError},

		{"rita import --rolling --chunk 0 --numchunks 24",
			exists, rolling, 11, 12, rolling, 0, 24, default12, !delete, cfg{12, rolling, 0, 24}, !returnsError},

		{"rita import --numchunks 24",
			exists, rolling, 11, 12, !rolling, blank, 24, default12, !delete, cfg{12, rolling, 12, 24}, !returnsError},

		{"rita import --chunk 5 (default 12)",
			exists, rolling, 11, 12, !rolling, 5, blank, default12, !delete, cfg{12, rolling, 5, 12}, !returnsError},

		{"rita import --chunk 12 (default 12)", // error reason: chunk must be less than db numchunks
			exists, rolling, 11, 12, !rolling, 12, blank, default12, !delete, cfg{}, returnsError},

		{"rita import --chunk 12 (default 24)", // error reason: chunk must be less than db numchunks
			exists, rolling, 11, 12, !rolling, 12, blank, default24, !delete, cfg{}, returnsError},

		{"rita import --chunk 12 --numchunks 24",
			exists, rolling, 11, 12, !rolling, 12, 24, default12, !delete, cfg{12, rolling, 12, 24}, !returnsError},

		{"rita import --delete (default 12)",
			exists, rolling, 11, 12, !rolling, blank, blank, default12, delete, cfg{12, rolling, 11, 12}, !returnsError},

		{"rita import --delete --rolling (default 12)",
			exists, rolling, 11, 12, !rolling, blank, blank, default12, delete, cfg{12, rolling, 11, 12}, !returnsError},

		{"rita import --delete --chunk 5 (default 12)",
			exists, rolling, 11, 12, !rolling, 5, blank, default12, delete, cfg{12, rolling, 5, 12}, !returnsError},

		{"rita import --delete --rolling --chunk 0 --numchunks 24 (default 12)",
			exists, rolling, 11, 12, rolling, 0, 24, default12, delete, cfg{12, rolling, 0, 24}, !returnsError},

		// rolling, current chunk 11, total chunks 24
		{"rita import",
			exists, rolling, 11, 24, !rolling, blank, blank, default12, !delete, cfg{12, rolling, 12, 24}, !returnsError},

		{"rita import --rolling",
			exists, rolling, 11, 24, rolling, blank, blank, default12, !delete, cfg{12, rolling, 12, 24}, !returnsError},

		{"rita import --rolling --chunk 0 --numchunks 24",
			exists, rolling, 11, 24, rolling, 0, 24, default12, !delete, cfg{12, rolling, 0, 24}, !returnsError},

		{"rita import --numchunks 12", // error reason: cannot reduce the number of chunks
			exists, rolling, 11, 24, !rolling, blank, 12, default12, !delete, cfg{}, returnsError},

		{"rita import --chunk 12 --numchunks 12", // error reason: cannot reduce the number of chunks
			exists, rolling, 11, 24, !rolling, 12, 12, default12, !delete, cfg{}, returnsError},

		{"rita import --chunk 13 (default 12)",
			exists, rolling, 11, 24, !rolling, 13, blank, default12, !delete, cfg{12, rolling, 13, 24}, !returnsError},

		{"rita import --delete (default 12)",
			exists, rolling, 11, 24, !rolling, blank, blank, default12, delete, cfg{12, rolling, 11, 24}, !returnsError},

		{"rita import --delete --rolling (default 12)",
			exists, rolling, 11, 12, !rolling, blank, blank, default12, delete, cfg{12, rolling, 11, 12}, !returnsError},

		{"rita import --delete --chunk 5 (default 12)",
			exists, rolling, 11, 24, !rolling, 5, blank, default12, !delete, cfg{12, rolling, 5, 24}, !returnsError},

		{"rita import --delete --rolling --chunk 0 --numchunks 24 (default 12)",
			exists, rolling, 11, 24, rolling, 0, 24, default12, !delete, cfg{12, rolling, 0, 24}, !returnsError},
	}

	// runner for the test table above
	for _, testCase := range testCases {
		actual, err := parseFlags(
			testCase.dbExists, testCase.dbIsRolling, testCase.dbCurrChunk, testCase.dbTotalChunks,
			testCase.userIsRolling, testCase.userCurrChunk, testCase.userTotalChunks, testCase.cfgDefaultChunks,
			testCase.deleteOldData,
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
			assert.NoErrorf(t, err, "db: <%s> cmd: <%s>", dbMsg, testCase.msg)
			assert.Equalf(t, testCase.expected, actual, "db: <%s> cmd: <%s>", dbMsg, testCase.msg)
		}
	}

}
