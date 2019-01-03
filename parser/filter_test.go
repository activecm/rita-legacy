package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterConnPair(t *testing.T) {

	fsTest := &FSImporter{
		res:             nil,
		indexingThreads: 1,
		parseThreads:    1,
		internal:        getParsedSubnets([]string{"8.8.8.8/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}),
		alwaysIncluded:  getParsedSubnets([]string{"8.8.8.8/32"}),
		neverIncluded:   getParsedSubnets([]string{"8.8.4.4/32"}),
	}

	type filterConnTest struct {
		src string
		dst string
		out bool
	}

	dbMetaInfos := []filterConnTest{
		filterConnTest{ // internal and internal
			src: "10.10.10.10",
			dst: "10.10.10.11",
			out: true,
		},
		filterConnTest{ // internal and internal
			src: "192.168.1.1",
			dst: "192.168.1.1",
			out: true,
		},
		filterConnTest{ // internal and internal
			src: "192.168.1.1",
			dst: "192.168.2.1",
			out: true,
		},
		filterConnTest{ // internal and always include
			src: "8.8.8.8",
			dst: "192.168.2.1",
			out: false,
		},
		filterConnTest{ // internal and always include
			src: "192.168.2.1",
			dst: "8.8.8.8",
			out: false,
		},
		filterConnTest{ //src and dst on opposing lists
			src: "8.8.8.8",
			dst: "8.8.4.4",
			out: false,
		},
		filterConnTest{ //src and dst on opposing lists
			src: "8.8.4.4",
			dst: "8.8.8.8",
			out: false,
		},
		filterConnTest{ // external and external
			src: "24.10.10.10",
			dst: "34.10.10.11",
			out: true,
		},
		filterConnTest{ // external and external
			src: "139.130.4.5",
			dst: "208.67.222.222",
			out: true,
		},
	}

	for _, testCase := range dbMetaInfos {
		output := fsTest.filterConnPair(testCase.src, testCase.dst)
		require.Equal(t, testCase.out, output)
	}
}
