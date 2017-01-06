#!/usr/bin/env bash

set -o errexit
set -o pipefail

echo -e "\e[34mRunning 'go get github.com/ocmdev/rita'

"
go get github.com/ocmdev/rita
cd $GOPATH/src/github.com/ocmdev/rita

echo -e "
\e[34mDone! Now we just have to build and install RITA.\e[0m"
go build
go install

echo "Dependencies should be all installed! Make sure to run 'sudo ./install.sh' in ~/go/src/github.com/ocmdev/rita
to complete the RITA installation!

Make sure you also configure Bro and run with 'sudo broctl deploy' and make sure MongoDB is running with the command 'mongo' or 'sudo mongo'.

Happy Hunting!"
