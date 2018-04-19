# Releases

Steps for creating a RITA release. 

- Tag a commit on master as a release
	- Checkout the commit
	- "git tag \[version\]"
		- Follow [SemVer](https://semver.org)
	- Push the tag to github using "git push origin \[version\]"
- Wait for Quay.io to build the docker image
- [Use docker to create the build](https://github.com/activecm/rita/blob/master/docs/Docker%20Usage.md#using-docker-to-build-rita)
	- Instead of "rita:master", use "rita:\[version\]"
- Go to the [releases](https://github.com/activecm/rita/releases) page
	- Click "Draft a new release"
	- Select the new "\[version\]" tag
	- Fill out the title and description with recent changes
		- If the config.yaml file changed, give a thorough description of the needed changes
	- Attach the following files:
		- The rita binary, pulled from the docker image
		- The rita.yaml file for the tagged code base
			- IMPORTANT: This must be named config.yaml in the release
		- The LICENSE file for the tagged code base
		- The install.sh file for the tagged code base
	- Publish the release
