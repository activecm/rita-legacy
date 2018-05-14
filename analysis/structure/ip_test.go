package structure

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPv4ToBinary(t *testing.T) {
	max := net.IPv4(255, 255, 255, 255)
	min := net.IPv4(0, 0, 0, 0)
	med := net.IPv4(128, 128, 128, 128)
	diff := net.IPv4(1, 2, 3, 4)

	maxInt := ipv4ToBinary(max)
	minInt := ipv4ToBinary(min)
	medInt := ipv4ToBinary(med)
	diffInt := ipv4ToBinary(diff)
	assert.Equal(t, int64(1)<<32-1, maxInt)
	assert.Equal(t, int64(0), minInt)
	assert.Equal(t, int64(128)<<24+128<<16+128<<8+128, medInt)
	assert.Equal(t, int64(1)<<24+2<<16+3<<8+4, diffInt)
}

func TestIPv6ToBinary(t *testing.T) {
	max := net.ParseIP("FFFF:FFFF:FFFF:FFFF:FFFF:FFFF:FFFF:FFFF")
	min := net.ParseIP("0000:0000:0000:0000:0000:0000:0000:0000")
	diff := net.ParseIP("1234:5678:9ABC:DEF0:0FED:CBA9:8765:4321")

	maxInt1, maxInt2, maxInt3, maxInt4 := ipv6ToBinary(max)
	assert.Equal(t, int64(1)<<32-1, maxInt1)
	assert.Equal(t, int64(1)<<32-1, maxInt2)
	assert.Equal(t, int64(1)<<32-1, maxInt3)
	assert.Equal(t, int64(1)<<32-1, maxInt4)

	minInt1, minInt2, minInt3, minInt4 := ipv6ToBinary(min)
	assert.Equal(t, int64(0), minInt1)
	assert.Equal(t, int64(0), minInt2)
	assert.Equal(t, int64(0), minInt3)
	assert.Equal(t, int64(0), minInt4)

	diffInt1, diffInt2, diffInt3, diffInt4 := ipv6ToBinary(diff)

	diffExpInt1, _ := strconv.ParseInt("12345678", 16, 64)
	assert.Equal(t, diffExpInt1, diffInt1)

	diffExpInt2, _ := strconv.ParseInt("9ABCDEF0", 16, 64)
	assert.Equal(t, diffExpInt2, diffInt2)

	diffExpInt3, _ := strconv.ParseInt("0FEDCBA9", 16, 64)
	assert.Equal(t, diffExpInt3, diffInt3)

	diffExpInt4, _ := strconv.ParseInt("87654321", 16, 64)
	assert.Equal(t, diffExpInt4, diffInt4)
}
