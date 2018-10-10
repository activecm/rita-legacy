package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrateDBMetaInfo(t *testing.T) {
	type migrationTest struct {
		in  DBMetaInfo
		out DBMetaInfo
	}

	dbMetaInfos := []migrationTest{
		migrationTest{
			DBMetaInfo{
				Name:           "Before ImportFinished Was Introduced In v1.1.0",
				ImportFinished: false,
				Analyzed:       false,
				ImportVersion:  "v1.0.99",
				AnalyzeVersion: "v1.0.99",
			},
			DBMetaInfo{
				Name:           "Before ImportFinished Was Introduced In v1.1.0",
				ImportFinished: true,
				Analyzed:       false,
				ImportVersion:  "v1.0.99",
				AnalyzeVersion: "v1.0.99",
			},
		},
	}

	for _, testCase := range dbMetaInfos {
		output, err := migrateDBMetaInfo(testCase.in)
		require.Nil(t, err)
		require.Equal(t, testCase.out, output)
	}
}
