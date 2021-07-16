module github.com/activecm/rita

go 1.14

// If urfave/cli is updated from v1.20.0 the corresponding autocomplete file
// should be updated in etc/bash_completion.d/rita
// https://github.com/urfave/cli/blob/master/autocomplete/bash_autocomplete

require (
	github.com/VividCortex/ewma v1.1.1 // indirect
	github.com/activecm/mgorus v0.1.1
	github.com/activecm/mgosec v0.1.1
	github.com/activecm/rita-bl v0.0.0-20200806232046-0db4a39fcf49
	github.com/blang/semver v3.5.1+incompatible
	github.com/creasty/defaults v1.3.0
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/golang/protobuf v1.3.3 // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/safebrowsing v0.0.0-20190214191829-0feabcc2960b // indirect
	github.com/google/uuid v1.1.2
	github.com/json-iterator/go v1.1.11
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/olekukonko/tablewriter v0.0.2-0.20190214164707-93462a5dfaa6
	github.com/pbnjay/memory v0.0.0-20201129165224-b12e5d931931
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5
	github.com/sirupsen/logrus v1.3.0
	github.com/skratchdot/open-golang v0.0.0-20190104022628-a2dfa6d0dab6
	github.com/stretchr/testify v1.3.0
	github.com/urfave/cli v1.20.0
	github.com/vbauerster/mpb v3.3.4+incompatible
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550 // indirect
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b // indirect
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637 // indirect
	gopkg.in/yaml.v2 v2.2.2
)
