susi-dev
========

## Basic Usage
```bash
susi-dev create cloud
susi-dev add cloud susi-core
susi-dev add cloud susi-duktape
susi-dev add cloud susi-gowebstack
susi-dev deploy cloud user@hostname.cloud.com

susi-dev create gateway
susi-dev add gateway susi-core
susi-dev add gateway susi-duktape
susi-dev add gateway susi-cluster --connect-to cloud --addr hostname.cloud.com
susi-dev deploy gateway user@gateway.address.com
```
