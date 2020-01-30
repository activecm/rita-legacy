# Contributing to RITA
---
## Want to help? We would love that!
Here are some ways to get involved, ranging in
difficulty from easiest to hardest

## Bug Hunting
Run the software and tell us when it breaks. We are happy to receive bug
reports

Just be sure to do the following:
* Check if the bug is already accounted for on the
[Github issue tracker](https://github.com/activecm/rita/issues)
  * If an issue already exists, add the relevant info in a comment
  * If not, create an issue and include the relevant info
* Give very specific descriptions of how to reproduce the bug
* Include the output of `rita --version`
* Include a description of your hardware (e.g. CPU, RAM, filesystems)
* Tell us about the size of the test and the physical resources available

## Contributing Code
There are several ways to contribute code to the RITA project.
Before diving in, follow the [Manual Installation Instructions](docs/Manual%20Installation.md)

* Work on bug fixes:
  * Find an issue you would like to work on in the Github tracker
  * Leave a comment letting us know you would like to work on it
* Add tests:
  * All too often code developed to meet milestones only undergoes
  empirical, human testing
  * We would love to see unit tests throughout RITA
  * There are a few sections of this project that currently have unit tests. [Here](https://github.com/activecm/rita/blob/master/analysis/beacon/analyzer_test.go) is a good example of an existing unit test.
  * Also when writing tests it is advisable to work backwards, start with what
  result you want to get and then work backwards through the code
  * When you're ready to test code run `go test ./...` from the root directory
  of the project
  * Feel free to refactor code to increase our ability to test it
* Add new features:
  * If you would like to become involved in the development effort, please hop
   on our [OFTC channel at #activecm](https://webchat.oftc.net/?channels=activecm)
   and chat about what is currently being worked on

### Running Static Tests
* Golint
  * Install [golint](https://github.com/golang/lint)
  * Run `golint ./... | grep -v '^vendor/'` from the root RITA directory
  * Fix any errors and run golint again to verify
* Gofmt
  * Run `gofmt -l . | grep -v '^vendor/'` from the root RITA directory to identify files containing styling errors
  * Run `gofmt -w .` to automatically resolve gofmt errors
* Go vet
  * Run `go tool vet $(find . -name '*.go' | grep -v '/vendor/')` from the root RITA directory
  * Fix any errors and run golint again to verify
* Go test
  * Run `go test -v -race ./...` from the root RITA directory
  * Ensure that all unit tests have passed

### Reviewing Automated Test Results
Automated tests are run against each commit on Travis CI. Build results may be viewed [here](https://travis-ci.org/activecm/rita).

### Gittiquette Summary
* In order to contribute to RITA, you must fork it
  * Do not `go get` or `git clone` your forked repo
  * Instead, `git remote set-url origin https://github.com/YOURGITHUBACCOUNT/rita` it to your existing forked RITA repository
* Split a branch off of master `git checkout -b [a-new-branch]`
* Push your commits to your remote if you wish to develop in the public
* When your work is finished, pull down the latest master branch, and rebase
your feature branch off of it
* Submit a pull request on Github

### Common Issues
* Building Rita using `go install` or `go build` yields a RITA version of `UNDEFINED`
  * Use `make` or `make install`
