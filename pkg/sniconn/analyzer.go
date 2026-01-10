package sniconn

import (
	"sync"

	"github.com/activecm/rita-legacy/config"
	"github.com/activecm/rita-legacy/database"
	"github.com/activecm/rita-legacy/pkg/data"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer records data regarding the connections between pairs of internal IP addresses and external FQDNs (SNIs)
	analyzer struct {
		chunk            int                        //current chunk (0 if not on rolling analysis)
		connLimit        int64                      // limit for strobe classification
		db               *database.DB               // provides access to MongoDB
		conf             *config.Config             // contains details needed to access MongoDB
		analyzedCallback func(database.BulkChanges) // called on each analyzed result
		closedCallback   func()                     // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *linkedInput          // holds unanalyzed data
		analysisWg       sync.WaitGroup             // wait for analysis to finish
	}
)

// newAnalyzer creates a new analyzer for recording sni connection records
func newAnalyzer(chunk int, connLimit int64, db *database.DB, conf *config.Config, analyzedCallback func(database.BulkChanges), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		connLimit:        connLimit,
		db:               db,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *linkedInput),
	}
}

// collect gathers unique connection records for analysis
func (a *analyzer) collect(datum *linkedInput) {
	a.analysisChannel <- datum
}

// close waits for the analyzer to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

// start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {

		for datum := range a.analysisChannel {

			var selector data.UniqueSrcFQDNPair
			if datum.TLS != nil {
				selector = datum.TLS.Hosts
			} else if datum.HTTP != nil {
				selector = datum.HTTP.Hosts
			}

			netNameUpdate := mainQuery(selector, a.chunk)
			tlsUpdate := tlsQuery(datum.TLS, datum.TLSZeekRecords, a.connLimit, a.chunk)
			httpUpdate := httpQuery(datum.HTTP, datum.HTTPZeekRecords, a.connLimit, a.chunk)

			totalUpdate := database.MergeBSONMaps(netNameUpdate, tlsUpdate, httpUpdate)

			a.analyzedCallback(database.BulkChanges{
				a.conf.T.Structure.SNIConnTable: []database.BulkChange{{
					Selector: selector.BSONKey(),
					Update:   totalUpdate,
					Upsert:   true,
				}},
			})

		}
		a.analysisWg.Done()
	}()
}

func mainQuery(selector data.UniqueSrcFQDNPair, chunk int) bson.M {
	return bson.M{
		"$set": bson.M{
			"src_network_name": selector.SrcNetworkName,
			"cid":              chunk,
		},
	}
}

func tlsQuery(datum *TLSInput, zeekRecords []*data.ZeekUIDRecord, strobeLimit int64, chunk int) bson.M {
	if datum == nil {
		return bson.M{}
	}

	// if this connection qualifies to be a strobe with the current number
	// of connections in the current datum, don't store bytes and ts.
	// it will not qualify to be downgraded to a beacon until this chunk is
	// outdated and removed. If only importing once - still just a strobe.
	ts := datum.Timestamps

	var bytes []int64
	var totalTwoWayBytes int64
	var totalDuration float64
	for _, zeekRecord := range zeekRecords {
		bytes = append(bytes, zeekRecord.Conn.OrigBytes)
		totalTwoWayBytes = totalTwoWayBytes + zeekRecord.Conn.OrigBytes + zeekRecord.Conn.RespBytes
		totalDuration += zeekRecord.Conn.Duration
	}

	isStrobe := datum.ConnectionCount >= strobeLimit
	if isStrobe {
		ts = []int64{}
		bytes = []int64{}
	}

	return bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"$each": []bson.M{{
					"cid": chunk,
					"tls": bson.M{
						"ts":        ts,
						"bytes":     bytes,
						"strobe":    isStrobe,
						"count":     datum.ConnectionCount,
						"tbytes":    totalTwoWayBytes,
						"tdur":      totalDuration,
						"dst_ips":   datum.RespondingIPs.Items(),
						"dst_ports": datum.RespondingPorts.Items(),

						"dst_cert_invalid": datum.RespondingCertInvalid,
						"subjects":         datum.Subjects.Items(),
						"ja3":              datum.JA3s.Items(),
						"ja3s":             datum.JA3Ss.Items(),
					},
				}},
			},
		},
	}

}

func httpQuery(datum *HTTPInput, zeekRecords []*data.ZeekUIDRecord, strobeLimit int64, chunk int) bson.M {
	if datum == nil {
		return bson.M{}
	}

	// if this connection qualifies to be a strobe with the current number
	// of connections in the current datum, don't store bytes and ts.
	// it will not qualify to be downgraded to a beacon until this chunk is
	// outdated and removed. If only importing once - still just a strobe.
	ts := datum.Timestamps

	var bytes []int64
	var totalTwoWayBytes int64
	var totalDuration float64
	for _, zeekRecord := range zeekRecords {
		bytes = append(bytes, zeekRecord.Conn.OrigBytes)
		totalTwoWayBytes = totalTwoWayBytes + zeekRecord.Conn.OrigBytes + zeekRecord.Conn.RespBytes
		totalDuration += zeekRecord.Conn.Duration
	}

	isStrobe := datum.ConnectionCount >= strobeLimit
	if isStrobe {
		ts = []int64{}
		bytes = []int64{}
	}

	return bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"$each": []bson.M{{
					"cid": chunk,
					"http": bson.M{
						"ts":        ts,
						"bytes":     bytes,
						"strobe":    isStrobe,
						"count":     datum.ConnectionCount,
						"tbytes":    totalTwoWayBytes,
						"tdur":      totalDuration,
						"dst_ips":   datum.RespondingIPs.Items(),
						"dst_ports": datum.RespondingPorts.Items(),

						"methods":     datum.Methods.Items(),
						"user_agents": datum.UserAgents.Items(),
					},
				}},
			},
		},
	}
}
