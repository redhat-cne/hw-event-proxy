## Running examples locally

The Hw Event Proxy works with [Cloud Event Proxy](https://github.com/redhat-cne/cloud-event-proxy).
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
sudo dnf install qpid-dispatch-router
qdrouterd &

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

Install the `Red Hat Integration - AMQ Interconnect` operator in a new namespace `<AMQP_NAMESPAVCE>` namespace from the OpenShift Web Console.

Open the `Red Hat Integration - AMQ Interconnect` operator, click `Create Interconnect` from the `Red Hat Integration - AMQ Interconnect` tab. Use default values and make sure the name is `amq-interconnect`.

Make sure amq-interconnect pods are running before the next step.
```shell
oc get pods -n `<AMQP_NAMESPAVCE>`
```

In consumer.yaml, change the `transport-host` args for `cloud-native-event-sidecar` container from
```
- "--transport-host=amqp://amq-interconnect"
```
to
```
- "--transport-host=amqp://amq-interconnect.<AMQP_NAMESPAVCE>.svc.cluster.local"
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
