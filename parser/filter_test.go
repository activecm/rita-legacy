package parser

import (
	"net"
	"testing"

	"github.com/activecm/rita/util"
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
		internal:        util.ParseSubnets([]string{"10.0.0.0/8"}),
		alwaysIncluded:  util.ParseSubnets([]string{"10.0.0.1/32", "10.0.0.3/32", "1.1.1.1/32", "1.1.1.3/32"}),
		neverIncluded:   util.ParseSubnets([]string{"10.0.0.2/32", "10.0.0.3/32", "1.1.1.2/32", "1.1.1.3/32"}),
	}

	// all permutations of being on internal, always, and never lists
	internal := "10.0.0.0"
	internalAlways := "10.0.0.1"
	internalNever := "10.0.0.2"
	internalAlwaysNever := "10.0.0.3"
	external := "1.1.1.0"
	externalAlways := "1.1.1.1"
	externalNever := "1.1.1.2"
	externalAlwaysNever := "1.1.1.3"

	testCases := []testCase{
		// internal to internal cases
		testCase{internal, internal, true, "internal to internal should be filtered"},
		testCase{internal, internalAlways, false, "AlwaysInclude should override internal to internal filter"},
		testCase{internal, internalNever, true, "NeverInclude should override internal to internal filter"},
		// one IP on both opposing lists => always takes precedent
		testCase{internal, internalAlwaysNever, false, "AlwaysInclude should override NeverInclude and internal to internal filter"},
		// src and dst on opposing lists => always takes precedent
		testCase{internalAlways, internalNever, false, "AlwaysInclude should override NeverInclude and internal to internal filter"},

		// internal to external cases
		testCase{internal, external, false, "internal to external should not be filtered"},
		testCase{internal, externalAlways, false, "AlwaysInclude should not be filtered"},
		testCase{internal, externalNever, true, "NeverInclude should override internal to external and be filtered"},
		testCase{internal, externalAlwaysNever, false, "AlwaysInclude should override NeverInclude when one IP is in both"},
		testCase{internalAlways, externalNever, false, "AlwaysInclude should override NeverInclude when src and dst conflict"},

		// external to internal cases
		testCase{external, internal, false, "external to internal should not be filtered"},
		testCase{external, internalAlways, false, "AlwaysInclude should not be filtered"},
		testCase{external, internalNever, true, "NeverInclude should override internal to external and be filtered"},
		testCase{external, internalAlwaysNever, false, "AlwaysInclude should override NeverInclude when one IP is in both"},
		testCase{externalAlways, internalNever, false, "AlwaysInclude should override NeverInclude when src and dst conflict"},

		// external to external cases
		testCase{external, external, true, "internal to internal should be filtered"},
		testCase{external, externalAlways, false, "AlwaysInclude should override external to external filter"},
		testCase{external, externalNever, true, "NeverInclude should override external to external filter"},
		testCase{external, externalAlwaysNever, false, "AlwaysInclude should override NeverInclude and external to external filter"},
		testCase{externalAlways, externalNever, false, "AlwaysInclude should override NeverInclude and external to external filter"},
	}

	for _, test := range testCases {
		output := fsTest.filterConnPair(net.ParseIP(test.src), net.ParseIP(test.dst))
		assert.Equal(t, test.out, output, test.msg)
	}
}

func TestFilterConnPairWithoutInternalSubnets(t *testing.T) {

	fsTest := &FSImporter{
		res:             nil,
		indexingThreads: 1,
		parseThreads:    1,
		// purposely omitting internal subnet definition
		alwaysIncluded: getParsedSubnets([]string{"10.0.0.1/32", "10.0.0.3/32", "1.1.1.1/32", "1.1.1.3/32"}),
		neverIncluded:  getParsedSubnets([]string{"10.0.0.2/32", "10.0.0.3/32", "1.1.1.2/32", "1.1.1.3/32"}),
	}

	// "internal" here is merely by convention as with no InternalSubnets
	// defined, these should not be treated differently from external
	internal := "10.0.0.0"
	internalNever := "10.0.0.2"
	external := "1.1.1.0"

	// only including test cases which differ from when InternalSubnets is defined
	testCases := []testCase{
		testCase{internal, internal, false, "internal to internal should not be filtered when InternalSubnets is empty"},
		// still apply the NeverInclude filter
		testCase{internal, internalNever, true, "NeverInclude should be applied even when InternalSubnets empty"},
		testCase{internal, external, false, "internal to external should not be filtered when InternalSubnets is empty"},
		testCase{external, internal, false, "external to internal should not be filtered when InternalSubnets is empty"},
		testCase{external, external, false, "external to external should not be filtered when InternalSubnets is empty"},
	}

	for _, test := range testCases {
		output := fsTest.filterConnPair(net.ParseIP(test.src), net.ParseIP(test.dst))
		assert.Equal(t, test.out, output, test.msg)
	}
}
