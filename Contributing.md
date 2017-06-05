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
[Github issue tracker](https://github.com/ocmdev/rita/issues)
  * If an issue already exists, add the following info in a comment
  * If not, create an issue, and include the following info
* Give very specific descriptions of how to reproduce the bug
  * Log files can be found at ~/.rita/logs
* Include the output of `rita --version`
* Include a description of your hardware (ex. CPU, RAM, filesystems)
* Tell us about the size of the test, and the physical resources available

## Contributing Code
There are several ways to contribute code to the RITA project.
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
  * Feel free to refactor code to increase our ability to test it
  * Join our [IRC](https://github.com/ocmdev/rita/wiki/RITA-Gittiquette) to
  learn more
* Add new features:
  * If you would like to become involved in the development effort, please hop
   on our [OFTC channel at #ocmdev](https://webchat.oftc.net/?channels=ocmdev)
   and chat about what is currently being worked on.

All of these tasks ultimately culminate in a pull request being issued,
reviewed, and merged. When interacting with RITA through Git please check out
the
[RITA Gittiquette page](https://github.com/ocmdev/rita/wiki/RITA-Gittiquette).
Go limits the ways you may use Git with an open source project such as RITA, so
it is important that you understand the procedures laid out here.

### Gittiquette Summary
* We currently have a dev and master branch on OCMDev
  * Master is our tagged release branch
  * Dev is our development and staging branch
  * As more users come to rely on RITA, we will introduce a release-testing branch
  for release candidates
* In order to contribute to RITA, you must fork it
  * Do not `go get` or `git clone` your forked repo
  * Instead, `git remote add` it to your existing RITA repository
* Checkout the dev branch `git checkout dev`
* Split a branch off of dev `git checkout -b [a-new-branch]`
* Push your commits to your remote if you wish to develop in the public
* When your work is finished, pull down the latest dev branch, and rebase
your feature branch off of it
* Submit a pull request on Github

### Switching to the `dev` Branch
* Install RITA using either the [installer](https://raw.githubusercontent.com/ocmdev/rita/master/install.sh) or
[manually](https://github.com/ocmdev/rita/wiki/Installation)
* `cd $GOPATH/src/github.com/ocmdev/rita`
* `git checkout dev`
* `make install`
* Configure a config file for the dev branch
  * Make a backup of your config file for the master branch
  * Copy over the config from `etc/rita.yaml` to `~/.rita/config.yaml`
  * Update the newly copied config to match your old one

### Common Issues
* Building Rita using `go install` or `go build` yields a RITA version of `UNDEFINED`
  * Use `make` or `make install`.
* The dev branch is likely to break compatibility with datasets processed using
the master branch
  * Usually, resetting analysis will take care of the incompatible datasets
  * If the parser has been altered, a fresh import may be needed
  * If errors persist, manually delete the MetaDB out of MongoDB
* The dev branch is likely to break compatibility with the default installed
config file at `~/.rita/config.yaml`
  * Make a backup of your config file for the master branch
  * Copy over the config from `etc/rita.yaml` to `~/.rita/config.yaml`
  * Update the newly copied config to match your old one
