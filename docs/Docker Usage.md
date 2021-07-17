# Docker Usage

You can run RITA using Docker! You have several options depending on your specific needs.
* [Running RITA with Docker Compose](#running-rita-with-docker-compose) - This is the simplest option and requires the least setup. You will have to provide your own Zeek logs.
* [Running RITA with Docker Using External Mongo](#running-rita-with-docker-using-external-mongo) - This option is useful if you do not want to use Docker Compose or you have an external Mongo server you wish to use.
* [Using Docker to Build RITA](#using-docker-to-build-rita) - You can use Docker to build a standalone RITA binary that runs on any Linux 64-bit CPU. This is useful if you want a portable binary but don't want to use Docker to actually run RITA.

## Obtaining the RITA Docker Image

The easiest way is to pull down the pre-built image.

```
docker pull quay.io/activecm/rita
```

You can also build the image from source.

```
docker build -t quay.io/activecm/rita .
```

## Running RITA with Docker Compose

You will need a config file where you have [put in your `InternalSubnets`](../Readme.md#configuration-file).
You will also need the path to your Zeek log files.

```
export CONIFG=/path/to/your/rita/config.yaml
export LOGS=/path/to/your/zeek/logs
docker-compose run --rm rita import /logs your-dataset
```

Note: If you'd like to use a specific version of RITA than the default `latest` you can do so using the `VERSION` variable.

```
export VERSION=v4.3.0
docker-compose run --rm rita --version
```

## Running RITA with Docker Using External Mongo

If you don't need/want the convenience of Docker Compose running the Mongo server for you, you can also use RITA without it. You will need to modify RITA's config file to point to your external Mongo server and invoke RITA like this:

```
docker run -it --rm \
	-v /path/to/your/zeek/logs:/logs:ro \
	-v /path/to/your/rita/config.yaml:/etc/rita/config.yaml:ro \
	quay.io/activecm/rita import /logs your-dataset
```

## Using Docker to Build RITA

You can use Docker to build a statically linked RITA binary for you. This binary should be portable between Linux 64-bit systems. Once you've obtained the RITA docker image (see the "Obtaining the RITA Docker Image" section above) you can run the following commands to copy the binary to your host system.

```
docker create --name rita quay.io/activecm/rita
docker cp rita:/rita ./rita
docker rm rita
```

Note that you will have to manually install the `config.yaml` files into `/etc/rita/` as well as create any directories referenced inside the `config.yaml` file.
