# Upgrading RITA

Updating between versions of RITA is normally straightforward. Most often all you need to do is download the newest RITA binary and replace the one on your system. You can also use the appropriate `install.sh` installer to update between versions. Download and run the `install.sh` file from the [releases page](https://github.com/activecm/rita/releases) that corresponds with the version of RITA you wish to install.

You may not need to update your config file at all as we include sane default settings for any new config value inside of RITA. This means that if your config file is missing a value, RITA will still have a default to use. However, if you need to customize any of these new values you'll have to update to the newer config file.

> :exclamation: **IMPORTANT** :exclamation:
> If you are upgrading to v2 from an earlier version you will need to modify your config file to include values for `Filtering: InternalSubnets`.

In some cases you may also need to update your config file to a newer version. You can always find the latest config file in [`etc/rita.yml`](https://github.com/activecm/rita/blob/master/etc/rita.yaml). If you use the `install.sh` script, the correct version of the config file will be downloaded for you to `/etc/rita/config.yaml.new`.

To update the config file, transfer over any values you customized in your existing config to the equivalent section of the new config. Then save a backup of your existing `/etc/rita/config.yaml` before you replace it with the new version.

Here are other useful tips for comparing differences between configs:
* Check the release notes for each of the versions of RITA for details on config file changes.
* Run `diff /etc/rita/config.yaml /etc/rita/config.yaml.new` to see a summary of both your customizations and any changes to the new config.
* Use `rita test-config` to see the config values RITA is using while it runs. This includes any default values set when your config file doesn't specify them. You can also specify a custom config file to further compare the differences like this: `rita test-config --config /etc/rita/config.yaml.new`.
