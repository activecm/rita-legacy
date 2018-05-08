#!/usr/bin/env bash
#
# This file tests *install.sh* on RITA's supported platforms
# Currently, RITA supports:
# - Ubuntu 14.04
# - Ubuntu 16.04
# - Security Onion (Not Tested Here)
# - CentOS 7
set -e

__err() {
  echo ""
  echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
  echo "INSTALLER TESTS FAILED"
  echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
}

trap '__err' ERR

_DOCKER_IMAGES="
quay.io/activecm/ubuntu-sudo:14.04
quay.io/activecm/ubuntu-sudo:16.04
quay.io/activecm/centos-sudo:7
"

for image in $_DOCKER_IMAGES; do
  echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
  echo "RUNNING $image"
  echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
  echo ""
  docker run --rm -it \
    -v $(pwd)/install.sh:/home/user/install.sh \
    $image /bin/bash -c "yes '' | sudo ./install.sh"
done
