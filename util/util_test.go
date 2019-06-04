package util

import (
	"math"
	// "os"
	// "path"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsIP(t *testing.T) {
	testIP := "1.1.1.1"
	notIP := "a.b.c.d"
	assert.True(t, IsIP(testIP))
	assert.False(t, IsIP(notIP))
}

// func TestFileExists(t *testing.T) {
// 	filePath := "./.jeinwei8380243unt4u"
// 	os.Remove(filePath)
// 	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0666)
// 	assert.Nil(t, err)
// 	file.Close()
// 	exists, err := Exists(filePath)
// 	assert.Nil(t, err)
// 	assert.True(t, exists)
// 	os.Remove(filePath)
// 	exists, err = Exists(filePath)
// 	assert.Nil(t, err)
// 	assert.False(t, exists)

// 	currBinary, err := os.Executable()
// 	assert.Nil(t, err)
// 	badPath := path.Join(currBinary, "non-existent-file")

// 	_, err = Exists(badPath)
// 	assert.NotNil(t, err)
// }

func TestSortByStringLength(t *testing.T) {
	strings := []string{"yy", "z", "aaaa"}
	sort.Sort(ByStringLength(strings))
	assert.Equal(t, "z", strings[0])
	assert.Equal(t, "yy", strings[1])
	assert.Equal(t, "aaaa", strings[2])
}

func TestSortableInt64(t *testing.T) {
	ints := []int64{3434, -1, -20, 0}
	sort.Sort(SortableInt64(ints))
	assert.Equal(t, int64(-20), ints[0])
	assert.Equal(t, int64(-1), ints[1])
	assert.Equal(t, int64(0), ints[2])
	assert.Equal(t, int64(3434), ints[3])
}

func TestAbs(t *testing.T) {
	max := int64(math.MaxInt64)
	pos := int64(1)
	zero := int64(0)
	neg := int64(-1)
	min := int64(math.MinInt64)

	assert.Equal(t, max, Abs(max))
	assert.Equal(t, pos, Abs(pos))
	assert.Equal(t, zero, Abs(zero))
	assert.Equal(t, -1*neg, Abs(neg))
	assert.Equal(t, -1*min, Abs(min))
}

func TestRound(t *testing.T) {
	negDown := -16.6
	negDownExp := int64(-17)
	negUp := -16.1
	negUpExp := int64(-16)
	posDown := 16.1
	posDownExp := int64(16)
	posUp := 16.6
	posUpExp := int64(17)
	assert.Equal(t, negDownExp, Round(negDown))
	assert.Equal(t, negUpExp, Round(negUp))
	assert.Equal(t, posDownExp, Round(posDown))
	assert.Equal(t, posUpExp, Round(posUp))
}

func TestMinMax(t *testing.T) {
	large := 100
	small := -100
	assert.Equal(t, large, Max(large, small))
	assert.Equal(t, large, Max(small, large))
	assert.Equal(t, small, Min(large, small))
	assert.Equal(t, small, Min(small, large))
}

func TestStringInSlice(t *testing.T) {
	tables := []struct {
		val  string
		list []string
		out  bool
	}{
		{"a", []string{"a", "b", "c", "d"}, true},
		{"abc", []string{"a", "b", "c", "d"}, false},
		{"ethan", []string{"ethan", "melissa"}, true},
		{"-1", []string{}, false},
		{"-1", []string{"-1"}, true},
		{"somethingsomething999", []string{"somethingsomething"}, false},
	}

	for _, test := range tables {
		output := StringInSlice(test.val, test.list)
		require.Equal(t, test.out, output)
	}

}
