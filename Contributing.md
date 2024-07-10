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
[Github issue tracker](https://github.com/activecm/rita-legacy/issues)
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
  * Find an issue you would like to work on in the Github tracker, especially [unassigned issues marked "good first issue"](https://github.com/activecm/rita-legacy/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22+no%3Aassignee)
  * Leave a comment letting us know you would like to work on it
* Add new features:
  * If you would like to become involved in the development effort, open a new issue or continue a discussion on an existing issue

### Running Static Tests
* You must have a RITA [development environment](https://github.com/activecm/rita-legacy/blob/master/docs/Manual%20Installation.md#installing-golang) set up and [golangci-lint](https://github.com/golangci/golangci-lint#install) installed to run the tests.
* Check the [Makefile](https://github.com/activecm/rita-legacy/blob/master/Makefile) for all options. Currently you can run `make test`, `make static-test`, and `make unit-test`. There is also `make integration-test` and docker variants that will require you install docker as well.

### Reviewing Automated Test Results
Automated tests are run against each pull request. Build results may be viewed [here](https://github.com/activecm/rita-legacy/actions).

### Gittiquette Summary
* In order to contribute to RITA, you must [fork it](https://github.com/activecm/rita-legacy/fork).
* Once you have a forked repo you will need to clone it to a very specific path which corresponds to _the original repo location_. This is due to the way packages are imported in Go programs.
  * `git clone [your forked repo git url]`
* Add `https://github.com/activecm/rita-legacy` as a new remote so you can pull new changes.
  * `git remote add upstream https://github.com/activecm/rita-legacy`
* Split a branch off of master .
  * `git checkout -b [your new feature]`
* When your work is finished, pull the latest changes from the upstream master and rebase your changes on it.
  * `git checkout master; git pull -r upstream master`
  * `git checkout [your new feature]; git rebase master`
* Push your commits to your repo and submit a pull request on Github.

Further info can be found in the [Gittiquette doc](docs/RITA%20Gittiquette.md) under the guidelines and contributors sections.

### Common Issues
* Building Rita using `go build` or `go install` yields a RITA version of `UNDEFINED`
  * Use `make` or `make install` instead
