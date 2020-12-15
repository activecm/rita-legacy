package commands

import (
	"strconv"
)

//helper functions for formatting floats and integers
func f(f float64) string {
	return strconv.FormatFloat(f, 'g', 6, 64)
}
func i(i int64) string {
	return strconv.FormatInt(i, 10)
}
