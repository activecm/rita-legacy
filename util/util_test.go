package util

import (
	"math"
	"os"
	"path"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIP(t *testing.T) {
	testIP := "1.1.1.1"
	notIP := "a.b.c.d"
	assert.True(t, IsIP(testIP))
	assert.False(t, IsIP(notIP))
}

func TestFileExists(t *testing.T) {
	filePath := "./.jeinwei8380243unt4u"
	os.Remove(filePath)
	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0666)
	assert.Nil(t, err)
	file.Close()
	exists, err := Exists(filePath)
	assert.Nil(t, err)
	assert.True(t, exists)
	os.Remove(filePath)
	exists, err = Exists(filePath)
	assert.Nil(t, err)
	assert.False(t, exists)

	currBinary, err := os.Executable()
	assert.Nil(t, err)
	badPath := path.Join(currBinary, "non-existant-file")

	_, err = Exists(badPath)
	assert.NotNil(t, err)
}

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

func TestRemoveSortedDuplicates(t *testing.T) {
	allSame := []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	allSameExp := []int64{0}
	normal := []int64{1, 2, 3, 3, 3, 4, 4, 4, 5, 6, 10}
	normalExp := []int64{1, 2, 3, 4, 5, 6, 10}
	allSameTest := RemoveSortedDuplicates(allSame)
	assert.ElementsMatch(t, allSameExp, allSameTest)
	normalTest := RemoveSortedDuplicates(normal)
	assert.ElementsMatch(t, normalExp, normalTest)
	assert.True(t, sort.IsSorted(SortableInt64(normalTest)))
}

func TestCountAndRemoveSortedDuplicates(t *testing.T) {
	allSame := []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	allSameExp := []int64{0}
	normal := []int64{1, 2, 3, 3, 3, 4, 4, 4, 5, 6, 10}
	normalExp := []int64{1, 2, 3, 4, 5, 6, 10}
	allSameTest, allSameCounts := CountAndRemoveSortedDuplicates(allSame)
	assert.ElementsMatch(t, allSameExp, allSameTest)
	assert.Equal(t, int64(len(allSame)), allSameCounts[0])

	normalTest, normalCounts := CountAndRemoveSortedDuplicates(normal)
	assert.ElementsMatch(t, normalExp, normalTest)
	assert.True(t, sort.IsSorted(SortableInt64(normalTest)))
	assert.Equal(t, int64(1), normalCounts[1])
	assert.Equal(t, int64(1), normalCounts[2])
	assert.Equal(t, int64(3), normalCounts[3])
	assert.Equal(t, int64(3), normalCounts[4])
	assert.Equal(t, int64(1), normalCounts[5])
	assert.Equal(t, int64(1), normalCounts[6])
	assert.Equal(t, int64(1), normalCounts[10])
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
