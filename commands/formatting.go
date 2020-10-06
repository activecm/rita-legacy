package commands

import (
	"github.com/activecm/rita/util"
	"net"
	"strconv"
)

//helper functions for formatting floats and integers
func f(f float64) string {
	return strconv.FormatFloat(f, 'g', 6, 64)
}
func i(i int64) string {
	return strconv.FormatInt(i, 10)
}

//helper function for formatting uniqueIP NetworkName
func validNetworkName(ip string, networkName *string) string {
	if networkName != nil {
		return *networkName
	}
	netIP := net.ParseIP(ip)
	if util.IPIsPubliclyRoutable(netIP) {
		return "Public"
	} else {
		return "Unknown Private Network"
	}
}
