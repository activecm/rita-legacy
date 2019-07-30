
```
#!/bin/sh

set -e

if ! [ $(id -u) = 0 ]; then
  echo "This installer must be run as root (sudo)"
  exit 1
fi

apt install cmake make gcc g++ flex bison libpcap-dev libssl-dev python-dev swig zlib1g-dev
apt install libcurl4-openssl-dev libprotobuf-dev
git clone https://github.com/actor-framework/actor-framework.git
cd actor-framework
git checkout tags/0.14.5
./configure
make
make test
make install
cd ..
apt install libgeoip-dev
wget http://geolite.maxmind.com/download/geoip/database/GeoLiteCity.dat.gz
gunzip GeoLiteCity.dat.gz
mv GeoLiteCity.dat /usr/share/GeoIP/GeoIPCity.dat
wget http://geolite.maxmind.com/download/geoip/database/GeoLiteCityv6-beta/GeoLiteCityv6.dat.gz
gunzip GeoLiteCityv6.dat.gz
mv GeoLiteCityv6.dat /usr/share/GeoIP/GeoIPCityv6.dat
apt install libgoogle-perftools-dev
wget http://www.read.seas.harvard.edu/~kohler/ipsumdump/ipsumdump-1.86.tar.gz
tar -xzf ipsumdump-1.86.tar.gz
cd ipsumdump-1.86
./configure
make
make install
cd ..
git clone --recursive git://git.bro.org/bro
cd bro
./configure
make
make install
#Add /usr/local/bro to PATH
```