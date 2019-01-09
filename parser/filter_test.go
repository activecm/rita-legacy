package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	src string
	dst string
	out bool
	msg string
}

func TestFilterConnPairWithInternalSubnets(t *testing.T) {

	fsTest := &FSImporter{
		res:             nil,
		indexingThreads: 1,
		parseThreads:    1,
		internal:        getParsedSubnets([]string{"10.0.0.0/8"}),
		alwaysIncluded:  getParsedSubnets([]string{"10.0.0.1/32", "10.0.0.3/32", "1.1.1.1/32", "1.1.1.3/32"}),
		neverIncluded:   getParsedSubnets([]string{"10.0.0.2/32", "10.0.0.3/32", "1.1.1.2/32", "1.1.1.3/32"}),
	}

	// all permutations of being on internal, always, and never lists
	internal :=              "10.0.0.0"
	internal_always :=       "10.0.0.1"
	internal_never :=        "10.0.0.2"
	internal_always_never := "10.0.0.3"
	external :=              "1.1.1.0"
	external_always :=       "1.1.1.1"
	external_never :=        "1.1.1.2"
	external_always_never := "1.1.1.3"

	testCases := []testCase{
		// internal to internal cases
		testCase{internal, internal, true, "internal to internal should be filtered"},
		testCase{internal, internal_always, false, "AlwaysInclude should override internal to internal filter"},
		testCase{internal, internal_never, true, "NeverInclude should override internal to internal filter"},
		// one IP on both opposing lists => always takes precedent
		testCase{internal, internal_always_never, false, "AlwaysInclude should override NeverInclude and internal to internal filter"},
		// src and dst on opposing lists => always takes precedent
		testCase{internal_always, internal_never, false, "AlwaysInclude should override NeverInclude and internal to internal filter"},

		// internal to external cases
		testCase{internal, external, false, "internal to external should not be filtered"},
		testCase{internal, external_always, false, "AlwaysInclude should not be filtered"},
		testCase{internal, external_never, true, "NeverInclude should override internal to external and be filtered"},
		testCase{internal, external_always_never, false, "AlwaysInclude should override NeverInclude when one IP is in both"},
		testCase{internal_always, external_never, false, "AlwaysInclude should override NeverInclude when src and dst conflict"},

		// external to internal cases
		testCase{external, internal, false, "external to internal should not be filtered"},
		testCase{external, internal_always, false, "AlwaysInclude should not be filtered"},
		testCase{external, internal_never, true, "NeverInclude should override internal to external and be filtered"},
		testCase{external, internal_always_never, false, "AlwaysInclude should override NeverInclude when one IP is in both"},
		testCase{external_always, internal_never, false, "AlwaysInclude should override NeverInclude when src and dst conflict"},

		// external to external cases
		testCase{external, external, true, "internal to internal should be filtered"},
		testCase{external, external_always, false, "AlwaysInclude should override external to external filter"},
		testCase{external, external_never, true, "NeverInclude should override external to external filter"},
		testCase{external, external_always_never, false, "AlwaysInclude should override NeverInclude and external to external filter"},
		testCase{external_always, external_never, false, "AlwaysInclude should override NeverInclude and external to external filter"},
	}

	for _, test := range testCases {
		output := fsTest.filterConnPair(test.src, test.dst)
		assert.Equal(t, test.out, output, test.msg)
	}
}

func TestFilterConnPairWithoutInternalSubnets(t *testing.T) {

	fsTest := &FSImporter{
		res:             nil,
		indexingThreads: 1,
		parseThreads:    1,
		// purposely omitting internal subnet definition
		alwaysIncluded:  getParsedSubnets([]string{"10.0.0.1/32", "10.0.0.3/32", "1.1.1.1/32", "1.1.1.3/32"}),
		neverIncluded:   getParsedSubnets([]string{"10.0.0.2/32", "10.0.0.3/32", "1.1.1.2/32", "1.1.1.3/32"}),
	}

	// "internal" here is merely by convention as with no InternalSubnets
	// defined, these should not be treated differently from external
	internal :=              "10.0.0.0"
	internal_never :=        "10.0.0.2"
	external :=              "1.1.1.0"

	// only including test cases which differ from when InternalSubnets is defined
	testCases := []testCase{
		testCase{internal, internal, false, "internal to internal should not be filtered when InternalSubnets is empty"},
		// still apply the NeverInclude filter
		testCase{internal, internal_never, true, "NeverInclude should be applied even when InternalSubnets empty"},
		testCase{internal, external, false, "internal to external should not be filtered when InternalSubnets is empty"},
		testCase{external, internal, false, "external to internal should not be filtered when InternalSubnets is empty"},
		testCase{external, external, false, "external to external should not be filtered when InternalSubnets is empty"},
	}

	for _, test := range testCases {
		output := fsTest.filterConnPair(test.src, test.dst)
		assert.Equal(t, test.out, output, test.msg)
	}
}
