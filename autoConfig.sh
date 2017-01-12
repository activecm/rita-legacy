#!/usr/bin/env bash

set -o errexit
set -o pipefail

__explanation() {
	cat <<HEREDOC
So here's what this script will need to do to prepare for RITA:

1) Download and install GNU Netcat, Bro, Golang, and the latest version of MongoDB.

The MongoDB, netcat and golang versions we'd like aren't a part of the regular Ubuntu apt packages, but this script will add the key to the latest MongoDB repo to your package manager and install/auto config it and everything else.

2) Set up a Golang development enviornment in order to 'go get' and 'build' RITA.

This requires us to create directory "go" in your home folder and add a new PATH and GOPATH entry to your .bashrc

HEREDOC
}

__install() {
  echo -e "

  \e[34mUpdating packages. You may be prompted for your password for sudo operations if you haven't been already...

  \e[0m"

  sleep 3s

  sudo apt update

  echo -e "

  \e[34mGreat! Now installing RITA dependencies...

  \e[0m"
  sleep 3s

  sudo apt install -y bro
  sudo apt install -y broctl
  sudo apt install -y build-essential

  # golang most recent update
  # Wish there was a permalink so we don't have to update this every time
  wget https://storage.googleapis.com/golang/go1.7.1.linux-amd64.tar.gz
  sudo tar -zxvf  go1.7.1.linux-amd64.tar.gz -C /usr/local/
  sudo rm go1.7.1.linux-amd64.tar.gz

  # gnu-netcat
  wget https://sourceforge.net/projects/netcat/files/netcat/0.7.1/netcat-0.7.1.tar.gz
  tar -zxf netcat-0.7.1.tar.gz
  rm netcat-0.7.1.tar.gz
  cd netcat-0.7.1
  ./configure
  sudo make
  sudo make install
  cd ..
  rm -rf netcat-0.7.1

  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

  echo -e "

  \e[34mDone! Now just need to configure Go dev environment...

  \e[0m"

  sleep 3s

  if [[ -z "${GOPATH}" ]];
  then
    mkdir -p $HOME/go/{src,pkg,bin}
    echo 'export GOPATH=$HOME/go' >> $HOME/.bashrc
    echo 'export PATH=$PATH:$GOPATH/bin' >> $HOME/.bashrc
  else
    echo -e "\e[34mGOPATH seems to be set, we'll skip this part then for now
    "
  fi


  echo -e "

  \e[34mNow we need to get package key and MongoDB package...

  \e[0m"

  sleep 3s

  sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv 0C49F3730359A14518585931BC711F9BA15703C6

  echo "deb [ arch=amd64,arm64 ] http://repo.mongodb.org/apt/ubuntu xenial/mongodb-org/3.4 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-3.4.list

  sudo apt update
  sudo apt install -y mongodb-org

  sudo mkdir -p /data/db
  sudo chown -R $USER /data


  echo -e "\e[34mMake sure to start the mongoDB service with 'sudo service mongod start'.

  \e[34mIf you need to stop Mongo at any time, run 'sudo service mongod stop'

  \e[34mYou can access the mongo shell with 'sudo mongo'

  \e[34mBro must also be configured with 'broctl deploy'

  \e[34mIn order to continue the installation, reload bash config with 'source ~/.bashrc' and then run 'sudo -E ./install.sh'

  \e[0m"
}

__entry() {
  __explanation

  echo "
  This script requires that you have sudo access...
  "
  sudo -v

  if [[ "${1:-}" =~ ^-h|--help$ ]]
  then
    exit
  else
    printf "Hey you got sudo privileges! Great! We can continue with the installation...\n\n"
  fi

  read -p "Start the auto config script?[Y/n]" -n 1 -r

  if [[ ! $REPLY =~ ^[Yy]$ ]]
  then
    printf "\nAborting\n"
    exit -1
  fi

  __install
}

__entry "${@:-}"
