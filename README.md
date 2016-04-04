# susi-dev
susi-dev is a tool to setup distributed susi projects easily.

## Commands

* susi-dev create node-name -> bootstrap a new node
* susi-dev add node-name component -> setup a component on the given node
* susi-dev deploy node-name target -> deploy a node to a target
* susi-dev pki create folder -> create a new public key infrastructure
* susi-dev pki add folder client-name -> create and sign a new client certificate

## Usage by example
```bash
## Create a cloud instance

susi-dev create cloud # setup a new instance named 'cloud'
susi-dev add cloud susi-core # add susi-core to 'cloud'
susi-dev add cloud susi-duktape # add susi-duktape (js interpreter) to 'cloud'
susi-dev add cloud susi-gowebstack # add susi-gowebstack (http server) to 'cloud'

# configure webstack config and add js sources

susi-dev deploy cloud user@hostname.cloud.com # deploy to hostname.cloud.com

## Create a gateway instance

susi-dev create gateway # setup a new instance named 'gateway'
susi-dev add gateway susi-core # add susi-core to 'cloud'
susi-dev add gateway susi-duktape # add susi-duktape to 'cloud'

# add susi-cluster to gateway and connect it to the cloud
susi-dev add gateway susi-cluster --connect-to cloud --addr hostname.cloud.com

susi-dev deploy gateway user@gateway.address.com # deploy to gateway.address.com
```
