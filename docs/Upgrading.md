# Upgrading RITA

> :exclamation: **IMPORTANT** :exclamation:
> If you are upgrading from a version prior to v2 you will need to modify your config file to include values for `Filtering: InternalSubnets`.

## Upgrading Between Major Versions

If you are upgrading across major versions (e.g. v1.x.x to v2.x.x, or v2.x.x to v3.x.x), you will need to delete your existing datasets and re-import them. Major version bumps typically bring massive performance gains as well as new features at the cost of removing compatibility for older datasets.

You will likely need to update your config file as well. See the [Updating RITA's Config File](#updating-ritas-config-file) section below.

## Upgrading Between Minor or Patch Versions

If you are upgrading within the same major version (e.g. v2.0.0 to v2.0.1, or v3.0.0 to v3.1.0) all you need to do is download the newest RITA binary and replace the one on your system. Alternatively, you can download and run the `install.sh` file from the [releases page](https://github.com/activecm/rita/releases) that corresponds with the version of RITA you wish to install.

You may not need to update your config file at all as RITA includes sane default settings for any new config value. This means that if your config file is missing a value, RITA will still have a default to use. However, if you need to customize any of these new values you'll have to update to the newer config file.

## Updating RITA's Config File

In some cases you may also need to update your config file to a newer version. You can always find the latest config file in [`etc/rita.yml`](https://github.com/activecm/rita/blob/master/etc/rita.yaml). If you use the `install.sh` script, the correct version of the config file will be downloaded for you to `/etc/rita/config.yaml.new`.

To update the config file, transfer over any values you customized in your existing config to the equivalent section of the new config. Then save a backup of your existing `/etc/rita/config.yaml` before you replace it with the new version.

Here are other useful tips for comparing differences between configs:
* Check the release notes for each of the versions of RITA for details on config file changes.
* Run `diff /etc/rita/config.yaml /etc/rita/config.yaml.new` to see a summary of both your customizations and any changes to the new config.
* Use `rita test-config` to see the config values RITA is using while it runs. This includes any default values set when your config file doesn't specify them. You can also specify a custom config file to further compare the differences like this: `rita test-config --config /etc/rita/config.yaml.new`. 