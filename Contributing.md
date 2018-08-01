# Contributing to RITA
---
## Want to help? We would love that!
Here are some ways to get involved, ranging in
difficulty from easiest to hardest.

## Bug Hunting
Run the software and tell us when it breaks. We are happy to receive bug
reports. This software was developed for internal use
on the fly as needed. This means that the code was not built to the
typical standards of an open source project, but we would like to see it get
there.

Just be sure to do the following:
* Check if the bug is already accounted for on the
[Github issue tracker](https://github.com/activecm/rita/issues)
  * If an issue already exists, add the following info in a comment
  * If not, create an issue, and include the following info
* Give very specific descriptions of how to reproduce the bug
  * Log files can be found at ~/.rita/logs
* Include the output of `rita --version`
* Include a description of your hardware (ex. CPU, RAM, filesystems)
* Tell us about the size of the test, and the physical resources available

## Contributing Code
There are several ways to contribute code to the RITA project.
Before diving in, follow the [Manual Installation Instructions](docs/Manual%20Installation.md)

* Add godoc comments and fix style compliance issues:
  * Run the [go metalinter](https://github.com/alecthomas/gometalinter)
  * Find a linting error and fix it
* Work on bug fixes:
  * Find an issue you would like to work on in the Github tracker
  * Leave a comment letting us know you would like to work on it
* Add tests:
  * All too often code is developed to meet milestones which only undergoes
  empirical, human testing
  * We would love to see unit tests throughout RITA
  * Currently we only have unit tests for Beacon check under analysis/beacon to
  see how tests can be written neatly and easily
  * Also when writing tests it is advisable to work backwards, start with what
  result you want to get and then work backwards through the code
  * When you're ready to test code run `go test ./...` from the root directory
  of the project
  * Feel free to refactor code to increase our ability to test it
* Add new features:
  * If you would like to become involved in the development effort, please hop
   on our [OFTC channel at #activecm](https://webchat.oftc.net/?channels=activecm)
   and chat about what is currently being worked on.

All of these tasks ultimately culminate in a pull request being issued,
reviewed, and merged. 

### Gittiquette Summary
* In order to contribute to RITA, you must fork it
  * Do not `go get` or `git clone` your forked repo
  * Instead, `git remote add` it to your existing RITA repository
* Split a branch off of master `git checkout -b [a-new-branch]`
* Push your commits to your remote if you wish to develop in the public
* When your work is finished, pull down the latest master branch, and rebase
your feature branch off of it
* Submit a pull request on Github

### Common Issues
* Building Rita using `go install` or `go build` yields a RITA version of `UNDEFINED`
  * Use `make` or `make install`.
