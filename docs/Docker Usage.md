# Docker Usage

You can run RITA using Docker! You have several options depending on your specific needs.
* Running RITA with Docker Compose - This is the simplest option and requires the least setup. You will have to provide your own Bro logs.
* Running RITA with Docker Using External Mongo - This option is useful if you do not want to use Docker Compose or you have an external Mongo server you wish to use.
* Using Docker to Build RITA - You can use Docker to build a standalone RITA binary that runs on any Linux 64-bit CPU. This is useful if you want a portable binary but don't want to use Docker to actually run RITA.

## Obtaining the RITA Docker Image

The easiest way is to pull down the pre-built image.

```
docker pull activecm/rita:latest
```

You can also build the image from scratch.

```
docker build -t activecm/rita:latest .
```

## Running RITA with Docker Compose

At the very least, you will have to provide RITA with the path to your Bro log files using the `BRO_LOGS` environment variable.

```
export BRO_LOGS=/path/to/your/logs
docker-compose run --rm rita import
docker-compose run --rm rita analyze
```

You can also call it this way if you wish.

```
BRO_LOGS=/path/to/your/logs docker-compose run --rm rita import
BRO_LOGS=/path/to/your/logs docker-compose run --rm rita analyze
```

RITA will use the default `config.yaml` file which will work out of the box. If you wish to specify your own config file you can do so like this:

```
export BRO_LOGS=/path/to/your/logs
docker-compose run --rm -v /path/to/your/rita/config.yaml:/etc/rita/config.yaml rita show-databases
```

## Running RITA with Docker Using External Mongo

If you don't need/want the convenience of Docker Compose running the Mongo server for you, you can also use RITA without it. You will need to modify RITA's config file to point to your external Mongo server.

```
docker run -it --rm \
	-v /path/to/your/bro/logs:/opt/bro/logs/:ro \
	-v /path/to/your/rita/config.yaml:/etc/rita/config.yaml:ro \
	activecm/rita:latest import
docker run -it --rm \
	-v /path/to/your/bro/logs:/opt/bro/logs/:ro \
	-v /path/to/your/rita/config.yaml:/etc/rita/config.yaml:ro \
	activecm/rita:latest analyze
```

## Using Docker to Build RITA

You can use Docker to build a statically linked RITA binary for you. This binary should be portable between Linux 64-bit systems. Once you've obtained the RITA docker image (see the "Obtaining the RITA Docker Image" section above) you can run the following commands to copy the binary to your host system.

```
docker create --name rita activecm/rita:latest
docker cp rita:/rita ./rita
docker rm rita
```

Note that you will have to manually install the `config.yaml` and `tables.yaml` files into `/etc/rita/` as well as create any directories referenced inside the `config.yaml` file.
