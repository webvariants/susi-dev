# susi-dev
susi-dev is a tool to setup distributed susi projects easily.
It helps creating a PKI and adding config and systemd unit files for arbitary services.
susi-dev also automatically creates proper config file templates for the majority of susi services.

## Commands

* susi-dev create $node -> bootstrap a new node
* susi-dev add $node $component -> setup a component on the given node
* susi-dev deploy $node $target -> deploy a node to a target
* susi-dev pki create $folder -> create a new public key infrastructure
* susi-dev pki add $folder $client -> create and sign a new client certificate

## Hints

* $node is a unique node identifier like "cloud" or "host1"
* $component is a component name like "susi-core" or "vpn-server"
* $target is a username@host combination like "user@myhost.com"
* the user needs working sudo on the host

## Usage by example
```bash
## Create a cloud instance

susi-dev create cloud # setup a new instance named 'cloud'
susi-dev add cloud susi-core # add susi-core to 'cloud'
susi-dev add cloud susi-duktape # add susi-duktape (js interpreter) to 'cloud'
susi-dev add cloud susi-gowebstack # add susi-gowebstack (http server) to 'cloud'

# configure webstack config and add js sources...

susi-dev deploy cloud user@hostname.cloud.com # deploy to hostname.cloud.com

## Create a gateway instance

susi-dev create gateway # setup a new instance named 'gateway'
susi-dev add gateway susi-core # add susi-core to 'cloud'
susi-dev add gateway susi-duktape # add susi-duktape to 'cloud'

# add susi-cluster to gateway and connect it to the cloud
susi-dev add gateway susi-cluster --connect-to cloud --addr hostname.cloud.com

susi-dev deploy gateway user@gateway.address.com # deploy to gateway.address.com
```
