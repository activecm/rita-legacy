# Releases

Steps for creating a RITA release.

- Update the `install.sh` script so that the `_RITA_VERSION` variable reflects the desired version tag
	- Create a branch with this change and go through the pull request process
	- Ensure that all tests pass on this branch
	- Note: after merging this pull request, the install script will break until you complete the rest of these steps since the installer will pull the binary file from the release page on Github, which are both yet to be created

- Go to the [releases](https://github.com/activecm/rita/releases) page
	- Click `Draft a new release` or pick the already existing draft
	- Select the new `[version]` tag
	- Fill out the title and description with recent changes
		- If the config file changed, give a thorough description of the needed changes
	- Publish the release

- Wait for Quay.io to build the docker image
- [Use docker to create the build](https://github.com/activecm/rita/blob/master/docs/Docker%20Usage.md#using-docker-to-build-rita)
	- Instead of `rita:master`, use `rita:[version]`

- Go back to the release you published
	- Attach the following files:
		- The `rita` binary, pulled from the docker image
		- The `install.sh` file for the tagged code base