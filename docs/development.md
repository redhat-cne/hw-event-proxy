# Developer Guide

## Running examples locally

The Hardware Event Proxy works with [Cloud Event Proxy](https://github.com/redhat-cne/cloud-event-proxy).
Run cloud-event-proxy sidecar and consumer example from the cloud-event-proxy repo for testing locally.

### Set environment variables
```shell
export NODE_NAME=mynode
export HW_PLUGIN=true; export HW_EVENT_PORT=9087; export CONSUMER_TYPE=HW
export MSG_PARSER_PORT=9097; export MSG_PARSER_TIMEOUT=10
export LOG_LEVEL=debug
# replace the following with real Redfish credentials and BMC ip address
export REDFISH_USERNAME=admin; export REDFISH_PASSWORD=admin; export REDFISH_HOSTADDR=127.0.0.1

```

### Install and run Apache Qpid Dispach Router
```shell
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
scripts/build-go.sh
scripts/build-image.sh
TAG=xxx
podman push localhost/hw-event-proxy:${TAG} quay.io/redhat_emp1/hw-event-proxy:latest
```

## Deploying examples to kubernetes cluster

### Set Env variables
```shell
export VERSION=latest
export PROXY_IMG=quay.io/redhat_emp1/hw-event-proxy:${VERSION}
export SIDECAR_IMG=quay.io/redhat_emp1/cloud-event-proxy:${VERSION}
export CONSUMER_IMG=quay.io/redhat_emp1/cloud-native-event-consumer:${VERSION}
# replace the following with real Redfish credentials and BMC ip address
export REDFISH_USERNAME=admin; export REDFISH_PASSWORD=admin; export REDFISH_HOSTADDR=127.0.0.1
```

### Deploy for basic tests
```shell
make deploy-basic
```

### Undeploy
```shell
make undeploy-basic
```

## End to End Tests

Prerequisite: a working Kubernetes cluster. Have the environment variable `KUBECONFIG` set pointing to your cluster.

### Build Test Tool Image
```shell
cd e2e-tests
make build
scripts/build-image.sh
TAG=xxx
podman push localhost/redfish-event-test:${TAG} quay.io/redhat_emp1/redfish-event-test:latest
```

### Basic Test
```shell
make test
```
The basic test sets up one test pod and **one** consumer in the same node and sends out Redfish Events to the hw-event-proxy at a rate of 1 msg/sec.

The contents of the received events are verified in the test. The list of fields to check are defined the file [`e2e-tests/data/EVENT_FIELDS_TO_VERIFY`](../e2e-tests/data/EVENT_FIELDS_TO_VERIFY).

The events to be tested are defined in the `e2e-tests/data` folder with one JSON file per event. List of events are described [here](../e2e-tests/data/README.md).

#### Modify Tests
> 📝 Add new tests by adding JSON files in `e2e-tests/data`.
> 📝 Update message fields to check by updating `e2e-tests/data/EVENT_FIELDS_TO_VERIFY`.


### Performance Test
```shell
make test-perf
```
The basic test sets up one test pod and **20** consumers in the same node and sends out Redfish Events to the `hw-event-proxy` at a rate of 10 msgs/sec for 10 minutes.

The tests are marked PASSED if the performance targets are met.

Performance Target: **95%** of the massages should have latency within **10ms**.
