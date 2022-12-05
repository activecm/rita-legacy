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
	//sorter handles sorting the timestamp deltas and data sizes
	//of pairs of hosts in order to prepare the data for quantile based
	//statistical analysis
	sorter struct {
		db             *database.DB       // provides access to MongoDB
		conf           *config.Config     // contains details needed to access MongoDB
		sortedCallback func(*uconn.Input) // called on each sorted result
		closedCallback func()             // called when .close() is called and no more calls to sortedCallback will be made
		sortChannel    chan *uconn.Input  // holds unsorted data
		sortWg         sync.WaitGroup     // wait for analysis to finish
	}
)

// newSorter creates a new sorter which sorts unique connection data
// for use in quantile based statistics
func newSorter(db *database.DB, conf *config.Config, sortedCallback func(*uconn.Input), closedCallback func()) *sorter {
	return &sorter{
		db:             db,
		conf:           conf,
		sortedCallback: sortedCallback,
		closedCallback: closedCallback,
		sortChannel:    make(chan *uconn.Input),
	}
}

// collect gathers a chunk of data to be sorted
func (s *sorter) collect(data *uconn.Input) {
	s.sortChannel <- data
}

// close waits for the sorter to finish
func (s *sorter) close() {
	close(s.sortChannel)
	s.sortWg.Wait()
	s.closedCallback()
}

// start kicks off a new sorter thread
func (s *sorter) start() {
	s.sortWg.Add(1)
	go func() {

		for data := range s.sortChannel {
			if (data.TsList) != nil {
				//sort the size and timestamps to compute quantiles in the analyzer
				sort.Sort(util.SortableInt64(data.TsList))
				sort.Sort(util.SortableInt64(data.OrigBytesList))
			}
			s.sortedCallback(data)
		}
		s.sortWg.Done()
	}()
}
