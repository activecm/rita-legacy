package beacon

import (
	"sort"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/util"
)

type (
	sorter struct {
		db             *database.DB       // provides access to MongoDB
		conf           *config.Config     // contains details needed to access MongoDB
		sortedCallback func(*uconn.Input) // called on each analyzed result
		closedCallback func()             // called when .close() is called and no more calls to analyzedCallback will be made
		sortChannel    chan *uconn.Input  // holds unanalyzed data
		sortWg         sync.WaitGroup     // wait for analysis to finish
	}
)

//newsorter creates a new collector for gathering data
func newSorter(db *database.DB, conf *config.Config, sortedCallback func(*uconn.Input), closedCallback func()) *sorter {
	return &sorter{
		db:             db,
		conf:           conf,
		sortedCallback: sortedCallback,
		closedCallback: closedCallback,
		sortChannel:    make(chan *uconn.Input),
	}
}

//collect sends a chunk of data to be analyzed
func (s *sorter) collect(data *uconn.Input) {
	s.sortChannel <- data
}

//close waits for the collector to finish
func (s *sorter) close() {
	close(s.sortChannel)
	s.sortWg.Wait()
	s.closedCallback()
}

//start kicks off a new analysis thread
func (s *sorter) start() {
	s.sortWg.Add(1)
	go func() {

		for data := range s.sortChannel {

			if (data.TsList) != nil {
				//sort the size and timestamps since they may have arrived out of order
				sort.Sort(util.SortableInt64(data.TsList))
				sort.Sort(util.SortableInt64(data.OrigBytesList))

			}

			s.sortedCallback(data)

		}
		s.sortWg.Done()
	}()
}

//CountAndRemoveConsecutiveDuplicates removes consecutive
//duplicates in an array of integers and counts how many
//instances of each number exist in the array.
//Similar to `uniq -c`, but counts all duplicates, not just
//consecutive duplicates.
func countAndRemoveConsecutiveDuplicates(numberList []int64) ([]int64, map[int64]int64) {
	//Avoid some reallocations
	result := make([]int64, 0, len(numberList)/2)
	counts := make(map[int64]int64)

	last := numberList[0]
	result = append(result, last)
	counts[last]++

	for idx := 1; idx < len(numberList); idx++ {
		if last != numberList[idx] {
			result = append(result, numberList[idx])
		}
		last = numberList[idx]
		counts[last]++
	}
	return result, counts
}
