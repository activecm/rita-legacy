package structure

import (
	"encoding/binary"
	"net"
)

//ipv4ToBinary generates binary representations of the IPv4 addresses
func ipv4ToBinary(ipv4 net.IP) int64 {
	return int64(binary.BigEndian.Uint32(ipv4[12:16]))
}

// *** Note: for future ipv6 support *** //

// //ipv6ToBinary generates binary representations of the IPv6 addresses
// func ipv6ToBinary(ipv6 net.IP) structureTypes.IPv6Integers {
// 	ipv6Binary1 := int64(binary.BigEndian.Uint32(ipv6[0:4]))
// 	ipv6Binary2 := int64(binary.BigEndian.Uint32(ipv6[4:8]))
// 	ipv6Binary3 := int64(binary.BigEndian.Uint32(ipv6[8:12]))
// 	ipv6Binary4 := int64(binary.BigEndian.Uint32(ipv6[12:16]))
// 	return structureTypes.IPv6Integers{
// 		I1: ipv6Binary1,
// 		I2: ipv6Binary2,
// 		I3: ipv6Binary3,
// 		I4: ipv6Binary4,
// 	}
// }
