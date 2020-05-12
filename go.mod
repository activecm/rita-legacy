module github.com/activecm/rita

go 1.14

// If urfave/cli is updated from v1.20.0 the corresponding autocomplete file
// should be updated in etc/bash_completion.d/rita
// https://github.com/urfave/cli/blob/master/autocomplete/bash_autocomplete

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/VividCortex/ewma v1.1.1
	github.com/activecm/mgorus v0.1.1
	github.com/activecm/mgosec v0.1.1
	github.com/activecm/rita-bl v0.0.0-20180713181704-c067dd0a1359
	github.com/blang/semver v3.5.1+incompatible
	github.com/creasty/defaults v1.3.0
	github.com/davecgh/go-spew v1.1.1
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/golang/protobuf v1.3.3
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0
	github.com/google/safebrowsing v0.0.0-20190214191829-0feabcc2960b
	github.com/konsorten/go-windows-terminal-sequences v1.0.1
	github.com/mattn/go-isatty v0.0.4
	github.com/mattn/go-runewidth v0.0.4
	github.com/olekukonko/tablewriter v0.0.2-0.20190214164707-93462a5dfaa6
	github.com/pmezard/go-difflib v1.0.0
	github.com/rakyll/statik v0.1.7
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5
	github.com/sirupsen/logrus v1.3.0
	github.com/skratchdot/open-golang v0.0.0-20190104022628-a2dfa6d0dab6
	github.com/stretchr/testify v1.3.0
	github.com/urfave/cli v1.20.0
	github.com/vbauerster/mpb v3.3.4+incompatible
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	golang.org/x/sys v0.0.0-20190412213103-97732733099d
	golang.org/x/text v0.3.0
	golang.org/x/tools v0.0.0-20200511174955-01e0872ccf9a
	google.golang.org/genproto v0.0.0-20200511104702-f5ebc3bea380
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
	gopkg.in/urfave/cli.v1 v1.20.0
	gopkg.in/yaml.v2 v2.2.2
)
