package commands

import (
	"encoding/hex"
	"strconv"

	"github.com/globalsign/mgo/bson"
)

// helper functions for formatting floats and integers
func f(f float64) string {
	return strconv.FormatFloat(f, 'g', 6, 64)
}
func i(i int64) string {
	return strconv.FormatInt(i, 10)
}

func b(b bson.Binary) string {
	return hex.EncodeToString(b.Data)
}
