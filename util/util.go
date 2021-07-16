package util

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

//TimeFormat stores a correctly formatted timestamp
const TimeFormat string = "2006-01-02-T15:04:05-0700"

//DayFormat stores a correctly formatted timestamp for the day
const DayFormat string = "2006-01-02"

// Exists returns true if file or directory exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// IsDir returns true if argument is a directory
func IsDir(path string) bool {
	file, err := os.Stat(path)
	if err != nil {
		return false
	}
	if file.IsDir() {
		return true
	}
	return false
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

//MaxUint64 returns the larger of two 64 bit unsigned integers
func MaxUint64(a uint64, b uint64) uint64 {
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

//Int64InSlice returns true if the int64 is an element of the array
func Int64InSlice(value int64, list []int64) bool {
	for _, entry := range list {
		if entry == value {
			return true
		}
	}
	return false
}

const (
	day  = time.Minute * 60 * 24
	year = 365 * day
)

// FormatDuration properly prints a given time.Duration
// https://gist.github.com/harshavardhana/327e0577c4fed9211f65#gistcomment-2557682
func FormatDuration(d time.Duration) string {
	if d < day {
		return d.String()
	}

	var b strings.Builder

	if d >= year {
		years := d / year
		fmt.Fprintf(&b, "%dy", years)
		d -= years * year
	}

	days := d / day
	d -= days * day
	fmt.Fprintf(&b, "%dd%s", days, d)

	return b.String()
}
