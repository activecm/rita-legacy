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

type testCaseDomain struct {
	domain string
	out    bool
	msg    string
}

type testCaseIsProxyIP struct {
	ip  string
	out bool
	msg string
}

type testCaseSingleIP struct {
	ip  string
	out bool
	msg string
}

func TestCheckIfProxyServer(t *testing.T) {

	fsTest := &filter{
		httpProxyServers: util.ParseSubnets([]string{"1.1.1.1", "1.1.1.2/32", "1.2.0.0/16"}),
	}

	// all permutations for possible IP matches/non-matches
	singleIPNoCIDRFiltered := "1.1.1.1"
	singleIPCIDRFiltered := "1.1.1.2"
	cidrRangeFiltered := "1.2.1.1"
	singleIPNotFiltered := "1.3.1.1"

	testCases := []testCaseIsProxyIP{
		{singleIPNoCIDRFiltered, true, "IP should match single, non-CIDR notation Proxy IP entry"},
		{singleIPCIDRFiltered, true, "IP should match CIDR notation (/32) for single Proxy IP entry"},
		{cidrRangeFiltered, true, "IP should match CIDR notation (/16) for Proxy IP range entry"},
		{singleIPNotFiltered, false, "IP should not match any Proxy IP entries"},
	}

	for _, test := range testCases {
		output := fsTest.checkIfProxyServer(net.ParseIP(test.ip))
		assert.Equal(t, test.out, output, test.msg)
	}
}

func TestFilterConnPairWithInternalSubnets(t *testing.T) {

	fsTest := &filter{
		internal:       util.ParseSubnets([]string{"10.0.0.0/8"}),
		alwaysIncluded: util.ParseSubnets([]string{"10.0.0.1/32", "10.0.0.3/32", "1.1.1.1/32", "1.1.1.3/32"}),
		neverIncluded:  util.ParseSubnets([]string{"10.0.0.2/32", "10.0.0.3/32", "1.1.1.2/32", "1.1.1.3/32"}),
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
		{internal, internal, true, "internal to internal should be filtered"},
		{internal, internalAlways, false, "AlwaysInclude should override internal to internal filter"},
		{internal, internalNever, true, "NeverInclude should override internal to internal filter"},
		// one IP on both opposing lists => always takes precedent
		{internal, internalAlwaysNever, false, "AlwaysInclude should override NeverInclude and internal to internal filter"},
		// src and dst on opposing lists => always takes precedent
		{internalAlways, internalNever, false, "AlwaysInclude should override NeverInclude and internal to internal filter"},

		// internal to external cases
		{internal, external, false, "internal to external should not be filtered"},
		{internal, externalAlways, false, "AlwaysInclude should not be filtered"},
		{internal, externalNever, true, "NeverInclude should override internal to external and be filtered"},
		{internal, externalAlwaysNever, false, "AlwaysInclude should override NeverInclude when one IP is in both"},
		{internalAlways, externalNever, false, "AlwaysInclude should override NeverInclude when src and dst conflict"},

		// external to internal cases
		{external, internal, false, "external to internal should not be filtered"},
		{external, internalAlways, false, "AlwaysInclude should not be filtered"},
		{external, internalNever, true, "NeverInclude should override internal to external and be filtered"},
		{external, internalAlwaysNever, false, "AlwaysInclude should override NeverInclude when one IP is in both"},
		{externalAlways, internalNever, false, "AlwaysInclude should override NeverInclude when src and dst conflict"},

		// external to external cases
		{external, external, true, "internal to internal should be filtered"},
		{external, externalAlways, false, "AlwaysInclude should override external to external filter"},
		{external, externalNever, true, "NeverInclude should override external to external filter"},
		{external, externalAlwaysNever, false, "AlwaysInclude should override NeverInclude and external to external filter"},
		{externalAlways, externalNever, false, "AlwaysInclude should override NeverInclude and external to external filter"},
	}

	for _, test := range testCases {
		output := fsTest.filterConnPair(net.ParseIP(test.src), net.ParseIP(test.dst))
		assert.Equal(t, test.out, output, test.msg)
	}
}

func TestFilterConnPairWithoutInternalSubnets(t *testing.T) {

	fsTest := &filter{
		// purposely omitting internal subnet definition
		alwaysIncluded: util.ParseSubnets([]string{"10.0.0.1/32", "10.0.0.3/32", "1.1.1.1/32", "1.1.1.3/32"}),
		neverIncluded:  util.ParseSubnets([]string{"10.0.0.4/32", "10.0.0.3/32", "1.1.1.2/32", "1.1.1.3/32"}),
	}

	// "internal" here is merely by convention as with no InternalSubnets
	// defined, these should not be treated differently from external
	internal := "10.0.0.0"
	internalNever := "10.0.0.4"
	external := "1.1.1.0"

	// only including test cases which differ from when InternalSubnets is defined
	testCases := []testCase{
		{internal, internal, false, "internal to internal should not be filtered when InternalSubnets is empty"},
		// still apply the NeverInclude filter
		{internal, internalNever, true, "NeverInclude should be applied even when InternalSubnets empty"},
		{internal, external, false, "internal to external should not be filtered when InternalSubnets is empty"},
		{external, internal, false, "external to internal should not be filtered when InternalSubnets is empty"},
		{external, external, false, "external to external should not be filtered when InternalSubnets is empty"},
	}

	for _, test := range testCases {
		output := fsTest.filterConnPair(net.ParseIP(test.src), net.ParseIP(test.dst))
		assert.Equal(t, test.out, output, test.msg)
	}
}

func TestFilterDomain(t *testing.T) {

	fsTest := &filter{
		internal:             util.ParseSubnets([]string{"10.0.0.0/8"}),
		alwaysIncluded:       util.ParseSubnets([]string{"10.0.0.1/32", "10.0.0.3/32", "1.1.1.1/32", "1.1.1.3/32"}),
		neverIncluded:        util.ParseSubnets([]string{"10.0.0.2/32", "10.0.0.3/32", "1.1.1.2/32", "1.1.1.3/32"}),
		alwaysIncludedDomain: []string{"bad.com", "google.com", "*.myotherdomain.com"},
		neverIncludedDomain:  []string{"good.com", "google.com", "*.mydomain.com"},
	}

	// all permutations of being on internal, always, and never lists
	always := "bad.com"
	never := "good.com"
	alwaysNever := "google.com"
	wildcardNever := "a.mydomain.com"
	wildcardAlways := "a.myotherdomain.com"

	testCases := []testCaseDomain{
		{always, false, "AlwaysIncludeDomain should keep this domain from being filtered"},
		{never, true, "NeverIncludeDomain should filter this domain"},
		{alwaysNever, false, "NeverIncludeDomain should be ovverriden by AlwaysIncludeDomain"},
		{wildcardNever, true, "NeverIncludeDomain wildcard should filter this domain"},
		{wildcardAlways, false, "AlwaysIncludeDomain wildcard should keep this domain from being filtered"},
	}

	for _, test := range testCases {
		output := fsTest.filterDomain(test.domain)
		assert.Equal(t, test.out, output, test.msg)
	}
}

func TestFilterSingleIP(t *testing.T) {

	fsTest := &filter{
		// purposely omitting internal subnet definition
		alwaysIncluded: util.ParseSubnets([]string{"10.0.0.1/32", "10.0.0.3/32", "1.1.1.1/32", "1.1.1.3/32"}),
		neverIncluded:  util.ParseSubnets([]string{"10.0.0.4/32", "10.0.0.3/32", "1.1.1.2/32", "1.1.1.3/32"}),
	}

	// all possibilities for filtering single IP
	always := "10.0.0.1"
	never := "10.0.0.4"
	alwaysNever := "10.0.0.3"

	testCases := []testCaseSingleIP{
		{always, false, "AlwaysInclude IP should not be filtered"},
		// still apply the NeverInclude filter
		{never, true, "NeverInclude IP should be filtered"},
		{alwaysNever, false, "AlwaysInclude should take precedence over NeverInclude"},
	}

	for _, test := range testCases {
		output := fsTest.filterSingleIP(net.ParseIP(test.ip))
		assert.Equal(t, test.out, output, test.msg)
	}
}
