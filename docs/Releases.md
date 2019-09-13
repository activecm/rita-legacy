# Releases

Steps for creating a RITA release.

- Update the `install.sh` script so that the `_RITA_VERSION` variable reflects the desired version tag
	- Create a branch with this change and go through the pull request process
	- Ensure that all tests pass on this branch
	- Note: after merging this pull request, the master install script will break until the release files are built and uploaded.

- Go to the [releases](https://github.com/activecm/rita/releases) page
	- Click `Draft a new release` or pick the already existing draft
	- Select the new `[version]` tag
	- Fill out the title and description with recent changes
		- If the config file changed, give a thorough description of the needed changes
	- Publish the release

- Keep refreshing the release page until the `rita` binary and `install.sh` script are automatically added. You can keep an eye on the progress on the [actions page](https://github.com/activecm/rita/actions)
