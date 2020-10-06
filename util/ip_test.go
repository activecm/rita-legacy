package util

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

type ipBoolTestCase struct {
	ip  string
	out bool
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
