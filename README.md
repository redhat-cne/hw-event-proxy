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
export LOG_LEVEL=debug
# replace the following with real Redfish credentials and BMC ip address
export REDFISH_USERNAME=admin; export REDFISH_PASSWORD=admin; export REDFISH_HOSTADDR=127.0.0.1

```

### Install and run Apache Qpid Dispach Router
```
sudo dnf install qpid-dispatch-router
qdrouterd &
```
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

## End to End Tests

Prerequisite: a working Kubernetes cluster. Have the environment variable `KUBECONFIG` set pointing to your cluster.

### Build Test Tool Image
```
cd e2e-tests
make build
scripts/build-image.sh
podman images
TAG=xxx
podman push localhost/redfish-event-test:${TAG} quay.io/redhat_emp1/redfish-event-test:latest
```

### Basic Test
The basic test sets up one test pod and **one** consumer in the same node and sends out Redfish Events to the hw-event-proxy at a rate of 1 msg/sec for 10 seconds.

```shell
make test
```
This invokes 2 test cases:
* TEST 1:  WITH MESSAGE FIELD
* TEST 2:  WITHOUT MESSAGE FIELD

NOTE: TEST 2 waits for a random duration between 1 to 60 seconds for preloading Redfish Registries. By making the wait time random the test is able to test different scenarios when the Message Parser is not, partly or fully ready to process event messages.

The tests are marked PASSED if all the events are received by the consumer. There is no verification of performance targets.

### Performance Test
The basic test sets up one test pod and **20** consumers in the same node and sends out Redfish Events to the hw-event-proxy at a rate of 10 msgs/sec for 10 minutes.

```shell
make test-perf
```
The tests are marked PASSED if all the events are received by the consumer and the performance targets are met.

Performance Target:

**95%** of the massages should have latency <= **10ms**.

### Test Report
Test Report is available at logs/_report.csv at end of the test run.