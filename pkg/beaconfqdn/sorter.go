package beaconfqdn

import (
	"sort"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/util"
)

type (
	sorter struct {
		db             *database.DB              // provides access to MongoDB
		conf           *config.Config            // contains details needed to access MongoDB
		sortedCallback func(*hostname.FqdnInput) // called on each analyzed result
		closedCallback func()                    // called when .close() is called and no more calls to analyzedCallback will be made
		sortChannel    chan *hostname.FqdnInput  // holds unanalyzed data
		sortWg         sync.WaitGroup            // wait for analysis to finish
	}
)

//newsorter creates a new collector for gathering data
func newSorter(db *database.DB, conf *config.Config, sortedCallback func(*hostname.FqdnInput), closedCallback func()) *sorter {
	return &sorter{
		db:             db,
		conf:           conf,
		sortedCallback: sortedCallback,
		closedCallback: closedCallback,
		sortChannel:    make(chan *hostname.FqdnInput),
	}
}

//collect sends a chunk of data to be analyzed
func (s *sorter) collect(entry *hostname.FqdnInput) {
	s.sortChannel <- entry
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
