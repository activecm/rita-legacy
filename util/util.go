package util

import (
	"math"
	"net"
	"os"
)

//TimeFormat stores a correctly formatted timestamp
const TimeFormat string = "2006-01-02-T15:04:05-0700"
//DayFormat stores a correctly formatted timestamp for the day
const DayFormat string = "2006-01-02"

// IsIP returns true if string is a valid IP address
func IsIP(ip string) bool {
	if net.ParseIP(ip) != nil {
		return true
	}
	return false
}

// Exists returns true if file or directory exists
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// ByStringLength Functions that, in combination with golang sort,
// allow users to sort a slice/list of strings by string length
// (shortest -> longest)
type ByStringLength []string

func (s ByStringLength) Len() int           { return len(s) }
func (s ByStringLength) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByStringLength) Less(i, j int) bool { return len(s[i]) < len(s[j]) }

// SortableInt64 functions that allow a golang sort of int64s
type SortableInt64 []int64

func (s SortableInt64) Len() int           { return len(s) }
func (s SortableInt64) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s SortableInt64) Less(i, j int) bool { return s[i] < s[j] }

//Abs returns two's complement 64 bit absolute value
func Abs(a int64) int64 {
	mask := a >> 63
	a = a ^ mask
	return a - mask
}

//Round returns rounded int64
func Round(f float64) int64 {
	return int64(math.Floor(f + .5))
}

//Min returns the smaller of two integers
func Min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

//Max returns the larger of two integers
func Max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

//StringInSlice returns true if the string is an element of the array
func StringInSlice(value string, list []string) bool {
	for _, entry := range list {
		if entry == value {
			return true
		}
	}
	return false
}
