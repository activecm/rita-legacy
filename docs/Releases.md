# Releases

Steps for creating a RITA release. 

- Tag a commit on master as a release
	- Checkout the commit
	- Tag the commit with `git tag [version]`
		- Follow [SemVer](https://semver.org)
	- Push the tag to github using `git push origin [version]`
- Wait for Quay.io to build the docker image
- [Use docker to create the build](https://github.com/activecm/rita/blob/master/docs/Docker%20Usage.md#using-docker-to-build-rita)
	- Instead of `rita:master`, use `rita:[version]`
- Go to the [releases](https://github.com/activecm/rita/releases) page
	- Click `Draft a new release`
	- Select the new `[version]` tag
	- Fill out the title and description with recent changes
		- If the config file changed, give a thorough description of the needed changes
	- Attach the following files:
		- The `rita` binary, pulled from the docker image
		- The `install.sh` file for the tagged code base
	- Publish the release
