package parser

import (
	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestNewUniqueIP(t *testing.T) {
	ip, err := newUniqueIP(net.ParseIP("192.168.1.1"), "ff0d0776-0cdc-4a10-b793-522bcd48a560", "test")
	assert.Nil(t, err, "no error on local ip with valid data")
	assert.Equal(t, "192.168.1.1", ip.IP, "ip correctly assigned on local ip with valid data")
	assert.Equal(t, bson.BinaryUUID, ip.NetworkUUID.Kind, "uuid kind set for local ip with valid data")
	assert.Equal(t, []byte{
		0xff, 0x0d, 0x07, 0x76,
		0x0c, 0xdc, 0x4a, 0x10,
		0xb7, 0x93, 0x52, 0x2b,
		0xcd, 0x48, 0xa5, 0x60,
	}, ip.NetworkUUID.Data, "uuid binary correctly parsed for local ip with valid data")
	assert.Equal(t, "test", *ip.NetworkName)

	ip, err = newUniqueIP(net.ParseIP("192.168.1.1"), "", "")
	assert.Equal(t, ErrNoAgentInfoSupplied, err, "return error for local ip with invalid data")
	assert.Equal(t, "192.168.1.1", ip.IP, "ip correctly assigned on local ip with invalid data")

	ip, err = newUniqueIP(net.ParseIP("192.168.1.1"), "invalid-uuid-here", "test")
	assert.NotNil(t, err, "return uuid parsing error for local ip with invalid uuid")
	assert.NotEqual(t, ErrNoAgentInfoSupplied, "return different errors for no uuid vs bad uuid")
	assert.Nil(t, ip.NetworkName, "don't set network name when uuid invalid for local ip")
	assert.Nil(t, ip.NetworkUUID, "don't set network uuid with uuid invalid for local ip")

	ip, err = newUniqueIP(net.ParseIP("8.8.8.8"), "", "")
	assert.Nil(t, err, "error should be nil when parsing public ip")
	assert.Equal(t, "8.8.8.8", ip.IP, "ip correctly assigned on public ip")
	assert.Nil(t, ip.NetworkName, "don't set network name for public ip")
	assert.Nil(t, ip.NetworkUUID, "don't set network uuid for public ip")

}
