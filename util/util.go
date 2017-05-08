package util

import (
	"math"
	"net"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const URL string = `^((ftp|https?):\/\/)?(\S+(:\S*)?@)?((([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))|(([a-zA-Z0-9]+([-\.][a-zA-Z0-9]+)*)|((www\.)?))?(([a-z\x{00a1}-\x{ffff}0-9]+-?-?)*[a-z\x{00a1}-\x{ffff}0-9]+)(?:\.([a-z\x{00a1}-\x{ffff}]{2,}))?))(:(\d{1,5}))?((\/|\?|#)[^\s]*)?$`

var rxURL = regexp.MustCompile(URL)
var GOOD = 0
var BAD = -1

//UNUSED
/*
 * Name:     IsURL
 * Purpose:  Returns true if string is a URL, false otherwise
 * comments:
 */
func IsURL(str string) bool {
	_, fail := strconv.ParseFloat(str, 64)
	if fail == nil {
		return false
	}

	if str == "" || len(str) <= 3 || strings.HasPrefix(str, ".") {
		return false
	}

	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	if strings.HasPrefix(u.Host, ".") {
		return false
	}
	if u.Host == "" && (u.Path != "" && !strings.Contains(u.Path, ".")) {
		return false
	}
	if net.ParseIP(str) != nil {
		return false
	}

	return rxURL.MatchString(str)
}

//UNUSED
/*
 * Name:     IsIP
 * Purpose:  Returns true if string is a valid IP address, false otherwise
 * comments:
 */
func IsIP(ip string) bool {
	if net.ParseIP(ip) != nil {
		return true
	}
	return false
}

//UNUSED
/*
 * Name:     IsLoopback
 * Purpose:  Returns true if string is a valid IP address, false otherwise
 * comments:
 */
func IsLoopback(ip string) bool {

	ip_parsed := net.ParseIP(ip)
	if ip_parsed == nil {
		return false
	}

	return ip_parsed.IsLoopback()
}

//UNUSED
// ValidIP validates an ip
func ValidIP(ip string) bool {
	if net.ParseIP(ip) != nil {
		return true
	}
	return false
}

// RFC1918 returns true if it the ip looks non routable
func RFC1918(ip string) bool {
	// if !ValidIP(ip) {
	// 	return false
	// }
	// // Attempt to filter loopback
	// if ip == "127.0.0.1" {
	// 	return true
	// }

	// octets := strings.Split(ip, ".")
	// if octets[0] == "10" {
	// 	return true
	// }

	// // This was already confirmed valid IP v4 octets[1] is an int
	// oct, _ := strconv.Atoi(octets[1])
	// if octets[0] == "172" && oct < 32 && oct > 15 {
	// 	return true
	// }
	// if octets[0] == "192" && oct == 168 {
	// 	return true
	// }

	// return false

	res := false

	if !IsIP(ip) {
		return res
	}

	octs := strings.Split(ip, ".")

	finalNum := ""

	for _, octCurr := range octs {
		for len(octCurr) < 3 {
			octCurr = "0" + octCurr
		}

		finalNum = finalNum + octCurr
	}

	ipInt, _ := strconv.Atoi(finalNum)

	switch {
	case (10000000000 <= ipInt) && (ipInt <= 10255255255),
		(172016000000 <= ipInt) && (ipInt <= 172031255255),
		(192168000000 <= ipInt) && (ipInt <= 192168255255):
		res = true
	}

	return res
}

//UNUSED
// IsSpecialIP attempts to filter some IPs that we don't care about
func IsSpecialIP(ip string) bool {
	v := net.ParseIP(ip)
	if v == nil {
		return false
	}
	if v.IsUnspecified() ||
		v.IsGlobalUnicast() ||
		v.IsInterfaceLocalMulticast() ||
		v.IsLinkLocalUnicast() ||
		v.IsMulticast() ||
		v.IsLinkLocalMulticast() {
		return true
	}
	return false
}

//UNUSED
/*
 * Name:     Exists
 * Purpose:  Returns true if file or directory exists, false otherwise
 * comments:
 */
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

//UNUSED
// Mean calculating the mean (average) of a data set
func Mean(numList []float64) float64 {
	total := 0.0
	for _, entry := range numList {
		total += entry
	}
	return total / float64(len(numList))
}

//UNUSED
// Variance calculating the variance of a data set
func Variance(numList []float64) float64 {
	mean := Mean(numList)
	varList := make([]float64, len(numList))
	for index, num := range numList {
		varList[index] = math.Pow((num - mean), 2)
	}
	return Mean(varList)
}

//UNUSED
// StdDev  calculating the standard deviation (quantified amount of variation
// or dispersion) of a data set
func StdDev(numList []float64) float64 {

	return math.Sqrt(Variance(numList))
}

//UNUSED
// AvgMaxPos calculating mean (average), maximum value, and position of
// max value in a data set
func AvgMaxPos(vals []float64) (avg float64, max float64, maxPos float64) {
	max = math.Inf(-1)
	maxPos = 0
	avg = 0.0

	for idx, val := range vals {
		avg += val

		if max < val {
			max = val
			maxPos = float64(idx)
		}
	}

	avg = avg / float64(len(vals))

	return
}

//UNUSED
/*
 * Name:     TypeConvert
 * Purpose:  Dynamic type converter
 * comments:
 */
func TypeConvert(m interface{}, desiredType reflect.Kind) (interface{}, int) {
	currType := reflect.TypeOf(m).Kind()
	if currType == desiredType {
		return m, GOOD
	}

	switch desiredType {
	default:
		return m, BAD
	case reflect.String:
		if currType == reflect.Int {
			return (strconv.Itoa(m.(int))), GOOD
		} else if currType == reflect.Int64 {
			return (strconv.FormatInt(m.(int64), 10)), GOOD
		} else if currType == reflect.Float64 {
			return (strconv.FormatFloat(m.(float64), 'f', -1, 64)), GOOD
		} else if currType == reflect.Float32 {
			return (strconv.FormatFloat(float64(m.(float32)), 'f', -1, 32)), GOOD
		} else if currType == reflect.Uint {
			return (strconv.FormatInt(m.(int64), 10)), GOOD
		}
	case reflect.Int:
		if currType == reflect.String {
			mTemp, err := strconv.Atoi(m.(string))
			if err != nil {
				//fmt.Println("Error parsing variable")
				return -1, BAD
			}
			return (mTemp), GOOD
		} else if currType == reflect.Float64 {
			return (int(m.(float64))), GOOD
		} else if currType == reflect.Int64 {
			return (int(m.(int64))), GOOD
		}
	case reflect.Int64:
		if currType == reflect.String {
			mTemp, err := strconv.Atoi(m.(string))
			if err != nil {
				//fmt.Println("Error parsing variable")
				return int64(-1), BAD
			}
			return (int64(mTemp)), GOOD
		} else if currType == reflect.Float64 {
			return (int64(m.(float64))), GOOD
		} else if currType == reflect.Int {
			return (int64(m.(int))), GOOD
		}
	case reflect.Float64:
		if currType == reflect.String {
			mTemp, err := strconv.ParseFloat(m.(string), 64)
			if err != nil {
				//fmt.Println("Error parsing variable")
				return float64(-1), BAD
			}
			return (mTemp), GOOD
		} else if currType == reflect.Int {
			return (float64(m.(int))), GOOD
		} else {
			//fmt.Println("Unrecognized Type!")
			return float64(-1), BAD
		}

	}

	return m, BAD
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

//given a sorted slice, remove the duplicates
func RemoveSortedDuplicates(sortedIn []int64) []int64 {
	//Avoid some reallocations
	result := make([]int64, 0, len(sortedIn)/2)
	last := sortedIn[0]
	result = append(result, last)

	for idx := 1; idx < len(sortedIn); idx++ {
		if last != sortedIn[idx] {
			result = append(result, sortedIn[idx])
		}
		last = sortedIn[idx]
	}
	return result
}

//two's complement 64 bit abs value
func Abs(a int64) int64 {
	mask := a >> 63
	a = a ^ mask
	return a - mask
}

//rounding function since go doesn't have it
func Round(f float64) int64 {
	return int64(math.Floor(f + .5))
}

//retun the smaller of two integers
func Min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
