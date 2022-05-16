package beaconfqdn

import (
	"sort"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/util"
)

type (
	//sorter handles sorting the timestamp deltas and data sizes
	//of (src, fqdn) pairs in order to prepare the data for quantile based
	//statistical analysis
	sorter struct {
		db             *database.DB     // provides access to MongoDB
		conf           *config.Config   // contains details needed to access MongoDB
		sortedCallback func(*fqdnInput) // called on each sorted result
		closedCallback func()           // called when .close() is called and no more calls to analyzedCallback will be made
		sortChannel    chan *fqdnInput  // holds unsorted data
		sortWg         sync.WaitGroup   // wait for analysis to finish
	}
)

//newsorter creates a new sorter which sorts (src->fqdn) connection data
//for use in quantile based statistics
func newSorter(db *database.DB, conf *config.Config, sortedCallback func(*fqdnInput), closedCallback func()) *sorter {
	return &sorter{
		db:             db,
		conf:           conf,
		sortedCallback: sortedCallback,
		closedCallback: closedCallback,
		sortChannel:    make(chan *fqdnInput),
	}
}

//collect sends a chunk of data to be sorted
func (s *sorter) collect(entry *fqdnInput) {
	s.sortChannel <- entry
}

//close waits for the sorter to finish
func (s *sorter) close() {
	close(s.sortChannel)
	s.sortWg.Wait()
	s.closedCallback()
}

//start kicks off a new sorter thread
func (s *sorter) start() {
	s.sortWg.Add(1)

	go func() {
		for entry := range s.sortChannel {
			if (entry.TsList) != nil {
				//sort the size and timestamps to compute quantiles in the analyzer
				sort.Sort(util.SortableInt64(entry.TsList))
				sort.Sort(util.SortableInt64(entry.OrigBytesList))
			}
			s.sortedCallback(entry)
		}
		s.sortWg.Done()
	}()
}
