package data

import (
	"net"
	"testing"

	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/assert"
)

func TestNewUniqueIP(t *testing.T) {
	ip := NewUniqueIP(net.ParseIP("192.168.1.1"), "ff0d0776-0cdc-4a10-b793-522bcd48a560", "test")
	assert.Equal(t, "192.168.1.1", ip.IP, "ip correctly assigned on private ip with valid data")
	assert.Equal(t, bson.BinaryUUID, ip.NetworkUUID.Kind, "uuid kind set for private ip with valid data")
	assert.Equal(t, []byte{
		0xff, 0x0d, 0x07, 0x76,
		0x0c, 0xdc, 0x4a, 0x10,
		0xb7, 0x93, 0x52, 0x2b,
		0xcd, 0x48, 0xa5, 0x60,
	}, ip.NetworkUUID.Data, "uuid binary correctly parsed for private ip with valid data")
	assert.Equal(t, "test", ip.NetworkName, "net name set for private ip with valid data")

	ip = NewUniqueIP(net.ParseIP("192.168.1.1"), "", "")
	assert.Equal(t, "192.168.1.1", ip.IP, "ip correctly assigned on private ip with no network data")
	assert.Equal(t, util.UnknownPrivateNetworkUUID.Kind, ip.NetworkUUID.Kind, "uuid kind set for private ip with no network data")
	assert.Equal(t, util.UnknownPrivateNetworkUUID.Data, ip.NetworkUUID.Data, "uuid binary set to flag value for private ip with no network data")
	assert.Equal(t, util.UnknownPrivateNetworkName, ip.NetworkName, "net name set to flag value for private ip with no network data")

	ip = NewUniqueIP(net.ParseIP("192.168.1.1"), "invalid-uuid-here", "test")
	assert.Equal(t, "192.168.1.1", ip.IP, "ip correctly assigned on private ip with invalid network data")
	assert.Equal(t, util.UnknownPrivateNetworkUUID.Kind, ip.NetworkUUID.Kind, "uuid kind set for private ip with invalid network data")
	assert.Equal(t, util.UnknownPrivateNetworkUUID.Data, ip.NetworkUUID.Data, "uuid binary set to flag value for private ip with invalid network data")
	assert.Equal(t, util.UnknownPrivateNetworkName, ip.NetworkName, "net name set to flag value for private ip with invalid network data")

	ip = NewUniqueIP(net.ParseIP("8.8.8.8"), "", "")
	assert.Equal(t, "8.8.8.8", ip.IP, "ip correctly assigned on public ip with no network data")
	assert.Equal(t, util.PublicNetworkUUID.Kind, ip.NetworkUUID.Kind, "uuid kind set for public ip with no network data")
	assert.Equal(t, util.PublicNetworkUUID.Data, ip.NetworkUUID.Data, "uuid binary set to flag value for public ip with no network data")
	assert.Equal(t, util.PublicNetworkName, ip.NetworkName, "net name set to flag value for public ip with no network data")

	ip = NewUniqueIP(net.ParseIP("8.8.8.8"), "invalid-uuid-here", "test")
	assert.Equal(t, "8.8.8.8", ip.IP, "ip correctly assigned on public ip with invalid network data")
	assert.Equal(t, util.PublicNetworkUUID.Kind, ip.NetworkUUID.Kind, "uuid kind set for public ip with invalid network data")
	assert.Equal(t, util.PublicNetworkUUID.Data, ip.NetworkUUID.Data, "uuid binary set to flag value for public ip with invalid network data")
	assert.Equal(t, util.PublicNetworkName, ip.NetworkName, "net name set to flag value for public ip with invalid network data")

	ip = NewUniqueIP(net.ParseIP("8.8.8.8"), "ff0d0776-0cdc-4a10-b793-522bcd48a560", "test")
	assert.Equal(t, "8.8.8.8", ip.IP, "ip correctly assigned on public ip with valid network data")
	assert.Equal(t, util.PublicNetworkUUID.Kind, ip.NetworkUUID.Kind, "uuid kind set for public ip with valid network data")
	assert.Equal(t, util.PublicNetworkUUID.Data, ip.NetworkUUID.Data, "uuid binary set to flag value for public ip with valid network data")
	assert.Equal(t, util.PublicNetworkName, ip.NetworkName, "net name set to flag value for public ip with valid network data")
}
