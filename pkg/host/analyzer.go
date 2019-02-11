package host

import (
	"encoding/binary"
	"net"
	"strings"
	"sync"

	"github.com/activecm/rita/pkg/uconn"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for host analysis
	analyzer struct {
		analyzedCallback func(update)    // called on each analyzed result
		closedCallback   func()          // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan uconn.Pair // holds unanalyzed data
		analysisWg       sync.WaitGroup  // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for gathering data
func newAnalyzer(analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan uconn.Pair),
	}
}

//collect sends a chunk of data to be analyzed
func (a *analyzer) collect(data uconn.Pair) {
	a.analysisChannel <- data
}

//close waits for the collector to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

//start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {

		for data := range a.analysisChannel {

			// update src of connection in hosts table
			if isIPv4(data.Src) {

				output := update{
					// create query
					query: bson.M{
						"$setOnInsert": bson.M{
							"local":                 data.IsLocalSrc,
							"ipv4":                  true,
							"ipv4_binary":           ipv4ToBinary(net.ParseIP(data.Src)),
							"max_beacon_score":      0.0,
							"max_beacon_conn_count": 0,
							"bl_out_count":          0,
							"bl_in_count":           0,
							"bl_sum_avg_bytes":      0,
							"bl_total_bytes":        0,
							"txt_query_count":       0,
						},
						"$inc": bson.M{
							"count_src": 1,
						},
						"$max": bson.M{"max_duration": data.MaxDuration},
					},
					// create selector for output
					selector: bson.M{"ip": data.Src},
				}

				// set to writer channel
				a.analyzedCallback(output)
			}

			// update dst of connection in hosts table
			if isIPv4(data.Dst) {

				output := update{
					// create query
					query: bson.M{
						"$setOnInsert": bson.M{
							"local":                 data.IsLocalDst,
							"ipv4":                  true,
							"ipv4_binary":           ipv4ToBinary(net.ParseIP(data.Dst)),
							"max_beacon_score":      0.0,
							"max_beacon_conn_count": 0,
							"bl_out_count":          0,
							"bl_in_count":           0,
							"bl_sum_avg_bytes":      0,
							"bl_total_bytes":        0,
							"txt_query_count":       0,
						},
						"$inc": bson.M{
							"count_dst": 1,
						},
						"$max": bson.M{"max_duration": data.MaxDuration},
					},
					// create selector for output
					selector: bson.M{"ip": data.Dst},
				}

				// set to writer channel
				a.analyzedCallback(output)
			}

		}
		a.analysisWg.Done()
	}()
}

//isIPv4 checks if an ip is ipv4
func isIPv4(address string) bool {
	return strings.Count(address, ":") < 2
}

//ipv4ToBinary generates binary representations of the IPv4 addresses
func ipv4ToBinary(ipv4 net.IP) int64 {
	return int64(binary.BigEndian.Uint32(ipv4[12:16]))
}
