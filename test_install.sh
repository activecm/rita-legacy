#!/usr/bin/env bash

mkdir -p install_results
sudo -v

set -x 

# Failure cases
# sudo docker run --rm -it -v `realpath ./install.sh`:/root/install.sh ubuntu:16.04 /root/install.sh &> install_results/ubuntu16_fail_new.txt
# sudo docker run --rm -it -v `realpath ./install.sh`:/root/install.sh ubuntu:18.04 /root/install.sh &> install_results/ubuntu18_fail_new.txt
# sudo docker run --rm -it -v `realpath ./install.sh`:/root/install.sh debian:10 /root/install.sh &> install_results/debian10_fail_new.txt
sudo docker run --rm -it -v `realpath ./install.sh`:/root/install.sh quay.io/centos/centos:stream /root/install.sh &> install_results/centosStream_new.txt

# # Success cases
# sudo docker run --rm -it -v `realpath ./install.sh`:/root/install.sh ubuntu:20.04 /root/install.sh &> install_results/ubuntu20_new.txt
# sudo docker run --rm -it -v `realpath ./install.sh`:/root/install.sh debian:11 /root/install.sh &> install_results/debian11_new.txt
# sudo docker run --rm -it -v `realpath ./install.sh`:/root/install.sh centos:7 /root/install.sh &> install_results/centos7_new.txt

# # Success cases with zeek disabled
# sudo docker run --rm -it -v `realpath ./install.sh`:/root/install.sh ubuntu:18.04 /root/install.sh --disable-zeek &> install_results/ubuntu18_new_no_zeek.txt
# sudo docker run --rm -it -v `realpath ./install.sh`:/root/install.sh debian:10 /root/install.sh --disable-zeek &> install_results/debian10_new_no_zeek.txt

# # Success cases with mongo disabled
# sudo docker run --rm -it -v `realpath ./install.sh`:/root/install.sh ubuntu:22.04 /root/install.sh --disable-mongo &> install_results/ubuntu22_new_no_mongo.txt


# # Run the old version

# # Failure cases
# sudo docker run --rm -it ubuntu:18.04 sh -c "apt update; apt install -y curl; curl -Lo /root/install.sh https://github.com/activecm/rita/releases/download/v4.6.0/install.sh; chmod +x /root/install.sh; /root/install.sh" &> install_results/ubuntu18_fail_old.txt
# sudo docker run --rm -it ubuntu:16.04 sh -c "apt update; apt install -y curl; curl -Lo /root/install.sh https://github.com/activecm/rita/releases/download/v4.6.0/install.sh; chmod +x /root/install.sh; /root/install.sh" &> install_results/ubuntu16_fail_old.txt
# sudo docker run --rm -it debian:10 sh -c "apt update; apt install -y curl; curl -Lo /root/install.sh https://github.com/activecm/rita/releases/download/v4.6.0/install.sh; chmod +x /root/install.sh; /root/install.sh" &> install_results/debian10_fail_old.txt
sudo docker run --rm -it quay.io/centos/centos:stream sh -c "yum install -y curl; curl -Lo /root/install.sh https://github.com/activecm/rita/releases/download/v4.6.0/install.sh; chmod +x /root/install.sh; /root/install.sh" &> install_results/centosStream_old.txt

# # Success cases
# sudo docker run --rm -it ubuntu:20.04 sh -c "apt update; apt install -y curl; curl -Lo /root/install.sh https://github.com/activecm/rita/releases/download/v4.6.0/install.sh; chmod +x /root/install.sh; /root/install.sh" &> install_results/ubuntu20_old.txt
# sudo docker run --rm -it debian:11 sh -c "apt update; apt install -y curl; curl -Lo /root/install.sh https://github.com/activecm/rita/releases/download/v4.6.0/install.sh; chmod +x /root/install.sh; /root/install.sh" &> install_results/debian11_old.txt
# sudo docker run --rm -it centos:7 sh -c "yum install -y curl; curl -Lo /root/install.sh https://github.com/activecm/rita/releases/download/v4.6.0/install.sh; chmod +x /root/install.sh; /root/install.sh" &> install_results/centos7_old.txt

# # Success cases with zeek disabled
# sudo docker run --rm -it ubuntu:18.04 sh -c "apt update; apt install -y curl; curl -Lo /root/install.sh https://github.com/activecm/rita/releases/download/v4.6.0/install.sh; chmod +x /root/install.sh; /root/install.sh --disable-zeek" &> install_results/ubuntu18_old_no_zeek.txt
# sudo docker run --rm -it debian:10 sh -c "apt update; apt install -y curl; curl -Lo /root/install.sh https://github.com/activecm/rita/releases/download/v4.6.0/install.sh; chmod +x /root/install.sh; /root/install.sh --disable-zeek" &> install_results/debian10_old_no_zeek.txt

# # Success cases with mongo disabled
# sudo docker run --rm -it ubuntu:22.04 sh -c "apt update; apt install -y curl; curl -Lo /root/install.sh https://github.com/activecm/rita/releases/download/v4.6.0/install.sh; chmod +x /root/install.sh; /root/install.sh --disable-mongo" &> install_results/ubuntu22_old_no_mongo.txt