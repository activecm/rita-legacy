package beacon

// write writes the beacon analysis results to the database
func (t *Beacon) write() {
	session := t.res.DB.Session.Copy()
	defer session.Close()

	for data := range t.writeChannel {
		session.DB(t.db).C(t.res.Config.T.Beacon.BeaconTable).Insert(data)
	}
	t.writeWg.Done()
}
