package structure

import (
	"net"
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

// *** Note: for future ipv6 support *** //

// func TestIPv6ToBinary(t *testing.T) {
// 	max := net.ParseIP("FFFF:FFFF:FFFF:FFFF:FFFF:FFFF:FFFF:FFFF")
// 	min := net.ParseIP("0000:0000:0000:0000:0000:0000:0000:0000")
// 	diff := net.ParseIP("1234:5678:9ABC:DEF0:0FED:CBA9:8765:4321")
//
// 	maxInt := ipv6ToBinary(max)
// 	assert.Equal(t, int64(1)<<32-1, maxInt.I1)
// 	assert.Equal(t, int64(1)<<32-1, maxInt.I2)
// 	assert.Equal(t, int64(1)<<32-1, maxInt.I3)
// 	assert.Equal(t, int64(1)<<32-1, maxInt.I4)
//
// 	minInt := ipv6ToBinary(min)
// 	assert.Equal(t, int64(0), minInt.I1)
// 	assert.Equal(t, int64(0), minInt.I2)
// 	assert.Equal(t, int64(0), minInt.I3)
// 	assert.Equal(t, int64(0), minInt.I4)
//
// 	diffInt := ipv6ToBinary(diff)
//
// 	diffExpInt1, _ := strconv.ParseInt("12345678", 16, 64)
// 	assert.Equal(t, diffExpInt1, diffInt.I1)
//
// 	diffExpInt2, _ := strconv.ParseInt("9ABCDEF0", 16, 64)
// 	assert.Equal(t, diffExpInt2, diffInt.I2)
//
// 	diffExpInt3, _ := strconv.ParseInt("0FEDCBA9", 16, 64)
// 	assert.Equal(t, diffExpInt3, diffInt.I3)
//
// 	diffExpInt4, _ := strconv.ParseInt("87654321", 16, 64)
// 	assert.Equal(t, diffExpInt4, diffInt.I4)
// }
