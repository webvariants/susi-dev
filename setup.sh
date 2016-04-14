#!/bin/bash

if uname -a | grep Ubuntu; then
  sudo apt-get --yes install golang gcc systemd-container rng-tools
fi

if uname -a | grep Debian; then
  sudo apt-get --yes install golang gcc rng-tools
  expected=3.18
  received=$(uname -r)
  min=$(echo -e $expected"\n"$received|sort -V|head -n 1)
  if ! [ "$min" = "$expected" ];then
    sudo bash -c 'echo "deb http://ftp.debian.org/debian jessie-backports main" > /etc/apt/sources.list.d/backports.list'
    sudo apt-get update
    sudo apt-get --yes -t jessie-backports install linux-image-amd64
    echo "We needed to update your kernel. Please reboot and run this script again."
    exit 0
  fi
fi

# setup go
mkdir ~/go
export GOPATH=~/go
export PATH=$GOPATH/bin:$PATH

# generate gpg key for container signing and export passphrase to current shell
if ! gpg --list-keys | grep pub; then
  rngd -r /dev/urandom
  gpg --gen-key
  gpg --export --armor > mykey.pub
fi
echo -n "Insert your gpg passphrase: "
read GPG_PASS
export GPG_PASS

# install susi-dev tool
go get github.com/webvariants/susi-dev

susi-dev setup
sudo rkt trust --root mykey.pub

bash

exit $?
