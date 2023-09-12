package util

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ipBoolTestCase struct {
	ip  string
	out bool
	msg string
}

type parseSubnetsTestCase struct {
	nets	[]string
	out []*net.IPNet
	wantErr bool
	msg string
}

func TestIPIsPublicRoutable(t *testing.T) {

	testCases := []ipBoolTestCase{
		{"10.1.2.3", false, "RFC1918 Class A"},
		{"172.16.1.2", false, "RFC1918 Class B"},
		{"192.168.1.2", false, "RFC1918 Class C"},
		{"fc00:1234::", false, "IPv6 local address"},
		{"127.0.0.5", false, "IPv4 loopback"},
		{"::1", false, "IPv6 loopback"},
		{"169.254.1.2", false, "IPv4 link local"},
		{"fe80:1234::", false, "IPv6 link local"},
		{"224.0.0.1", false, "IPv4 multicast"},
		{"ff12:1234::", false, "IPv6 multicast"},
		{"8.8.8.8", true, "google dns ipv4"},
		{"2001:4860:4860::8888", true, "google dns ipv6"},
	}

	for _, testCase := range testCases {
		output := IPIsPubliclyRoutable(net.ParseIP(testCase.ip))
		assert.Equal(t, testCase.out, output, testCase.msg)
	}
}

func TestIsIP(t *testing.T) {
	testIP := "1.1.1.1"
	notIP := "a.b.c.d"
	assert.True(t, IsIP(testIP))
	assert.False(t, IsIP(notIP))
}

// Ensures ParseSubnets returns expected net.IPNets and returns
// error when invalid IP address/CIDR network is provided.
func TestParseSubnets(t *testing.T) {
	validIPv4Nets := []string{"192.168.0.0/24", "10.0.0.0/16"}
	validIPv4NetsOut := parseCIDRs([]string{"192.168.0.0/24", "10.0.0.0/16"})
	validIPv4Hosts := []string{"192.168.0.1", "10.0.123.45"}
	validIPv4HostsOut := parseCIDRs([]string{"192.168.0.1/32", "10.0.123.45/32"})
	validIPv6Nets := []string{"2001:db8::/32", "2400:cb00:2048::/64"}
	validIPv6NetsOut := parseCIDRs([]string{"2001:db8::/32", "2400:cb00:2048::/64"})
	validIPv6Hosts := []string {"2001:db8::1", "2400:cb00:2048::1"}
	validIPv6HostsOut := parseCIDRs([]string {"2001:db8::1/128", "2400:cb00:2048::1/128"})
    invalidNets := []string{"invalidIP", "300.0.0.0/24"}

    testCases := []parseSubnetsTestCase{
        {
            nets:    validIPv4Nets,
            out:     validIPv4NetsOut,
            wantErr: false,
            msg:     "Valid IPv4 subnetworks",
        },
		{
            nets:    validIPv4Hosts,
            out:     validIPv4HostsOut,
            wantErr: false,
            msg:     "Valid IPv4 host IPs",
        },
		{
            nets:    validIPv6Nets,
            out:     validIPv6NetsOut,
            wantErr: false,
            msg:     "Valid IPv6 subnetworks",
        },
		{
            nets:    validIPv6Hosts,
            out:     validIPv6HostsOut,
            wantErr: false,
            msg:     "Valid IPv6 host IPs",
        },
        {
            nets:    invalidNets,
            out:     nil,
            wantErr: true,
            msg:     "Invalid IP and subnetwork (Expecting Error)",
        },
    }

    for _, testCase := range testCases {
        output, err := ParseSubnets(testCase.nets)
		if testCase.wantErr {
			assert.Error(t, err, testCase.msg)
		} else {
			assert.Equal(t, testCase.out, output, testCase.msg)
		}
    }
}


func parseCIDRs(cidr []string) []*net.IPNet {
    ipNets := make([]*net.IPNet, len(cidr))
	
    for i, ip := range cidr {
        _, ipNet, _ := net.ParseCIDR(ip)
        ipNets[i] = ipNet
    }

    return ipNets
}
