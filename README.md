# Hardware Event Proxy

Hardware Event Proxy handles Redfish hardware events. It contains a main `hw-event-proxy` module written in Go and a `message-parser` module written in Python.

The `message-parser` module is used to parse messages from Redfish Event Message Registry. At startup, it queries the Redfish API and downloads all the Message Registries (if not already included in Sushy library) including custom registries.

Once subscribed, Redfish events can be received by Webhook located in the `hw-event-proxy` module. If the event received does not contain the Message field, `hw-event-proxy` will send a request with Message ID to `message-parser`. Message Parser uses the Message ID to search in the Message Registries and find the Message and Resolution and pass them back to Hhw-event-proxy. `hw-event-proxy` adds them to the event content and converts the event to Cloud Event and sends it out to AMQP channel.  


## Running examples locally

The Hardware Event Proxy works with [Cloud Event Proxy](https://github.com/redhat-cne/cloud-event-proxy).
Run cloud-event-proxy sidecar and consumer example from the cloud-event-proxy repo for testing locally.

### Set environment variables
```
export NODE_NAME=mynode
export HW_PLUGIN=true; export HW_EVENT_PORT=9087; export CONSUMER_TYPE=HW
export MSG_PARSER_PORT=9097; export MSG_PARSER_TIMEOUT=10
export REDFISH_USERNAME=<redfish username>; export REDFISH_PASSWORD=<redfish password>; export REDFISH_HOSTADDR=<BMC host/ip address>
export LOG_LEVEL=debug
```

### Install and run Apache Qpid Dispach Router
Install AMQ router locally following https://github.com/redhat-cne/amq-installer.

### Run side car
```shell
cd <cloud-event-proxy repo>
make build-plugins
make run
```
### Run consumer
```shell
cd <cloud-event-proxy repo>
make run-consumer
```
### Run hw event proxy
```shell
cd <hw-event-proxy repo>/hw-event-proxy
make run
```
### Run message parser
```shell
cd <hw-event-proxy repo>/message-parser
# install dependencies
pip3 install -r requirements.txt
python3 server.py
```

## Building images

### Build with local dependencies

```shell
1. scripts/local-ldd-dep.sh
2. edit build-image.sh and rename Dockerfile to Dockerfile.local
```

### Build Images

```shell
1. scripts/build-go.sh
3. scripts/build-image.sh
# find out image tags ${TAG}
5. podman images
```

### Push images to a repo

```shell
podman push localhost/hw-event-proxy:${TAG} quay.io/redhat_emp1/hw-event-proxy:latest
```

Use consumer.yaml and service.yaml from examples/manifests folder to deploy to a cluster.
Make sure you update the image path.


## Deploying examples using kustomize

### Install Kustomize
```shell
curl -s "https://raw.githubusercontent.com/\
kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash

mv kustomize /usr/local/bin/

```
### Set Env variables
```shell
export VERSION=latest
export PROXY_IMG=quay.io/redhat_emp1/hw-event-proxy:${VERSION}
export SIDECAR_IMG=quay.io/redhat_emp1/cloud-event-proxy:${VERSION}
export CONSUMER_IMG=quay.io/redhat_emp1/cloud-native-event-consumer:${VERSION}
```

### Setup AMQ Interconnect

Install AMQ router following https://github.com/redhat-cne/amq-installer.

In consumer.yaml, change the `transport-host` args for `cloud-native-event-sidecar` container from
```
- "--transport-host=amqp://amq-interconnect"
```
to
```
- "--transport-host=amqp://router.router.svc.cluster.local"
```

### Set node affinity
The example consumer pod requires node affinity for baremetal worker node.
```
oc label node <worker node> app=local
```

### Deploy examples
```shell
make deploy-example
```

### Undeploy examples
```shell
make undeploy-example
```
