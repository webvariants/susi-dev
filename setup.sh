#!/bin/bash

sudo apt-get --yes install golang gcc systemd-container rng-tools

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
