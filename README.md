# susi-dev
susi-dev is a tool to setup distributed susi projects easily.
It helps creating a PKI and adding config and systemd unit files for arbitary services.
susi-dev also automatically creates proper config file templates for the majority of susi services. Additionally it can be used to test the setup in containers and to compile susi for different operating systems (debian stable, debian testing and alpine).

## Commands

* susi-dev create $node -> bootstrap a new node
* susi-dev add $node $component -> setup a component on the given node
* susi-dev deploy $node $target -> deploy a node to a target
* susi-dev source
  * clone -> clone the source of susi
  * checkout $branch -> checkout a specific branch
  * build --os $OS --gpgpass $pass -> build it for one of alpine, debian-stable, debian-testing or native
* susi-dev build $node --gpgpass $pass -> build containers for a node
* susi-dev start ($node) -> runs the containers
* susi-dev stop ($node) -> stops the containers
* susi-dev pki
  * create $folder -> create a new public key infrastructure
  * add $folder $client -> create and sign a new client certificate

## Hints

* $node is a unique node identifier like "cloud" or "host1"
* $component is a component name like "susi-core" or "vpn-server"
* $target is a username@host combination like "user@myhost.com"
* $branch is a valid susi branch
* $OS is one of alpine, debian-stable or debian-testing
* $pass is the passphrase of your users gpg key
* the user needs working sudo on the host

##  Getting started on Debian / Ubuntu
Execute this command on your machine and follow the instructions. If you are on Debian stable the script will update your kernel, so be prepared to reboot and re-run the script.
```bash
wget -qO /tmp/setup-susi-dev.sh https://raw.githubusercontent.com/webvariants/susi-dev/master/setup.sh && bash /tmp/setup-susi-dev.sh
```
Now susi-dev is fully setup and functional, and you have the GPG_PASS variable exported to your current shell.
Go ahead and paste the "How To Develop" code into your shell. After a few minutes you should see your first running susi container setup.

## How to develop

Simply copy and paste this into your console to simulate the setup of a gateway application.

```bash

# Download and compile susi binaries
susi-dev source build --gpgpass $GPG_PASS

# Create a cloud instance
susi-dev create gateway-1 # setup a new instance named 'gateway-1'
susi-dev add gateway-1 susi-core # add susi-core to 'gateway-1'
susi-dev add gateway-1 susi-duktape # add susi-duktape (js interpreter) to 'gateway-1'
susi-dev add gateway-1 susi-gowebstack # add susi-gowebstack (http server) to 'gateway-1'

# add js sources...
echo "\
susi.registerConsumer('duktape::ready',function(){\
  console.log('Hello World!');\
});\
susi.publish({topic: 'duktape::ready'});" > gateway-1/assets/duktape-script.js

# add ui
mkdir gateway-1/assets/webroot
echo "it works" > gateway-1/assets/webroot/index.html

# build containers for 'gateway-1'
susi-dev container build gateway-1 --gpgpass $GPG_PASS # build containers for 'gateway-1'
susi-dev container run gateway-1 # run containers for 'gateway-1'

```

Do you see the "Hello World!" in the logs? This comes from our deployed js.
And now get the IP of your container by listing all pods via "sudo rkt list"
and do a wget on it: "it works" ;)

## How to deploy

To deploy to a physical device or virtual machine, make sure you have deployed your ssh key to the machine (ssh-copy-id user@host) and you have sudo.
Then use the following command:
```bash
susi-dev deploy gateway-1 user@host
```
Now, your current configuration of 'gateway-1' is deployed to the machine 'host'.
Do not forget that you need to install the susi-binaries on that host. You can either copy the binaries by yourself, or deploy a matching debian package.

## How to build debian packages
If you want debian packages of susi, use the following commands.
```bash
susi-dev source build --os debian-stable --gpgpass $GPG_PASS
susi-dev source build --os debian-testing --gpgpass $GPG_PASS
```
Now the files susi-debian-stable.deb and susi-debian-testing.deb should be available in your working directory.
