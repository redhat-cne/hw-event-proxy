# Developer Guide

## Run Examples Locally

The Bare Metal Event Relay works with [Cloud Event Proxy](https://github.com/redhat-cne/cloud-event-proxy).
Run cloud-event-proxy sidecar and consumer example from the cloud-event-proxy repo for testing locally.

### Set Environment Variables
```shell
export NODE_NAME=mynode
export HW_PLUGIN=true; export HW_EVENT_PORT=9087; export CONSUMER_TYPE=HW
export MSG_PARSER_PORT=9097; export MSG_PARSER_TIMEOUT=10
export LOG_LEVEL=trace
# replace the following with real Redfish credentials and BMC ip address
export REDFISH_USERNAME=root; export REDFISH_PASSWORD=calvin; export REDFISH_HOSTADDR=123.123.123.123
```

### For AMQ Transport: Install And Run Apache Qpid Dispach Router
```shell
sudo dnf install qpid-dispatch-router
qdrouterd &
```

### Run SideCar
```shell
cd <cloud-event-proxy repo>
make build-plugins
# Test with HTTP Transport
go run cmd/main.go --transport-host="localhost:9043" --http-event-publishers="localhost:9043"
# Test with AMQ Transport
go run cmd/main.go --transport-host="amqp:localhost:5672"
```

### Run Consumer
```shell
cd <cloud-event-proxy repo>
make run-consumer
```

### Run Hw-event-proxy
```shell
cd <hw-event-proxy repo>/hw-event-proxy
make run
```

### Run Message Parser
```shell
cd <hw-event-proxy repo>/message-parser
# install dependencies
pip3 install -r requirements.txt
python3 server.py
```

### Send Events to Webhook
```shell

curl -X POST -i http://localhost:${HW_EVENT_PORT}/webhook -H "Content-Type: text/plain" --data @e2e-tests/data/TMP0100.json

# Test Message Parser
curl -X POST -i http://localhost:${HW_EVENT_PORT}/webhook -H "Content-Type: text/plain" --data @e2e-tests/data/TMP0100-no-msg-field.json
```

## Build Images

### Build With Local Dependencies

```shell
1. scripts/local-ldd-dep.sh
2. edit build-image.sh and rename Dockerfile to Dockerfile.local
```

### Build Images

```shell
scripts/build-go.sh
scripts/build-image.sh
TAG=xxx
podman push localhost/hw-event-proxy:${TAG} quay.io/jacding/hw-event-proxy:latest
```

## Deploy Examples To Kubernetes Cluster

### Set Env Variables
```shell
export VERSION=latest
export PROXY_IMG=quay.io/redhat-cne/hw-event-proxy:${VERSION}
export SIDECAR_IMG=quay.io/redhat-cne/cloud-event-proxy:${VERSION}
export CONSUMER_IMG=quay.io/redhat-cne/cloud-event-consumer:${VERSION}

# replace the following with real Redfish credentials and BMC ip address
export REDFISH_USERNAME=root; export REDFISH_PASSWORD=calvin; export REDFISH_HOSTADDR=123.123.123.123
```

### Deploy for Sanity Tests
```shell
# with HTTP Transport
make deploy

# with AMQ Transport
make deploy-amq
```

### Undeploy
```shell
# with HTTP Transport
make undeploy

# with AMQ Transport
make undeploy-amq
```

## End to End Tests

Prerequisite: a working Kubernetes cluster. Have the environment variable `KUBECONFIG` set pointing to your cluster.

### Build Test Tool Image
```shell
cd e2e-tests
make build
scripts/build-image.sh
TAG=xxx
podman push localhost/hw-event-proxy-e2e-test:${TAG} quay.io/jacding/hw-event-proxy-e2e-test:latest
```

### Sanity Test
```shell
# with HTTP Transport
make test

# with AMQ Transport
make test-amq
```
The sanity test sets up one test pod and **one** consumer in the same node and sends out Redfish Events to the hw-event-proxy at a rate of 1 msg/sec.

The contents of the received events are verified in the test. The list of fields to check are defined the file [`e2e-tests/data/EVENT_FIELDS_TO_VERIFY`](../e2e-tests/data/EVENT_FIELDS_TO_VERIFY).

The events to be tested are defined in the `e2e-tests/data` folder with one JSON file per event. List of events are described [here](../e2e-tests/data/README.md).

#### Modify Tests
> 📝 Add new tests by adding JSON files in `e2e-tests/data`. Add [description](../e2e-tests/data/README.md) if needed.

> 📝 Update message fields to check by updating `e2e-tests/data/EVENT_FIELDS_TO_VERIFY`.


### Performance Test
```shell
# with HTTP Transport
make test-perf

# with AMQ Transport
make test-perf-amq
```
The performance test sets up one test pod and **20** consumers in the same node and sends out Redfish Events to the `hw-event-proxy` at a rate of 10 msgs/sec for 10 minutes.

The tests are marked PASSED if the performance targets are met.

Performance Target: **95%** of the massages should have latency within **10ms**.

Full test report is available at ./logs/_report.csv

## Test with Curl Commands
### Create Redfish Subscription
```
curl -X POST -i --insecure -u "${REDFISH_USERNAME}:${REDFISH_PASSWORD}" https://${REDFISH_HOSTADDR}/redfish/v1/EventService/Subscriptions \
-H 'Content-Type: application/json' \
--data-raw '{
  "Protocol": "Redfish",
  "Context": "any string is valid",
  "Destination": "https://hw-event-proxy-openshift-hw-events.apps.example.com/webhook",
  "EventTypes": ["Alert"]
}'

# Create Redfish Subscription for ZT BMC
curl -X POST -i --insecure -u "${REDFISH_USERNAME}:${REDFISH_PASSWORD}" https://${REDFISH_HOSTADDR}/redfish/v1/EventService/Subscriptions \
-H 'Content-Type: application/json' \
--data-raw '{
  "Protocol": "Redfish",
  "Context": "any string is valid",
  "Destination": "https://hw-event-proxy-openshift-hw-events.apps.example.com/webhook"
}'

```

### List Redfish Subscriptions
```
curl --insecure -u "${REDFISH_USERNAME}:${REDFISH_PASSWORD}" https://${REDFISH_HOSTADDR}/redfish/v1/EventService/Subscriptions | jq .
```

### Delete Redfish Subscription
```
curl -X DELETE -i --insecure -u "${REDFISH_USERNAME}:${REDFISH_PASSWORD}" https://${REDFISH_HOSTADDR}/redfish/v1/EventService/Subscriptions/<sub_id>
```

### Submit a Test Event
```
curl -X POST -i --insecure -u "${REDFISH_USERNAME}:${REDFISH_PASSWORD}" https://${REDFISH_HOSTADDR}/redfish/v1/EventService/Actions/EventService.SubmitTestEvent \
-H 'Content-Type: application/json' \
--data-raw '{
  "EventId": "TestEventId",
  "EventTimestamp": "2022-08-23T15:13:49Z",
  "EventType": "Alert",
  "Message": "Test Event",
  "MessageId": "TMP0118",
  "OriginOfCondition": "/redfish/v1/Systems/1/",
  "Severity": "OK"
}'


# Submit Test Event for ZT BMC
curl -X POST -i --insecure -u "${REDFISH_USERNAME}:${REDFISH_PASSWORD}" https://${REDFISH_HOSTADDR}/redfish/v1/EventService/Actions/EventService.SubmitTestEvent \
-H 'Content-Type: application/json' \
--data-raw '{
  "MessageId": "EventLog.1.0.ResourceUpdated"
}'
```

### Send Redfish Events to Bare Metal Event Relay Directly
```
curl -X POST -i --insecure https://$(kubectl -n openshift-bare-metal-events get route hw-event-proxy -o jsonpath="{.spec.host}")/webhook \
  -H "Content-Type: text/plain" \
  --data @e2e-tests/data/TMP0100.json

```

## Deploy Example Consumer

### for 4.10:
```
export SIDECAR_IMG=quay.io/redhat-cne/cloud-event-proxy:release-4.10
export CONSUMER_IMG=quay.io/redhat-cne/cloud-event-consumer:release-4.10

# assuming amqp service is running at amq-router.amq-router
make deploy-consumer-amq
make undeploy-consumer-amq
```

### for 4.11 and later
```
export SIDECAR_IMG=quay.io/redhat-cne/cloud-event-proxy:latest
export CONSUMER_IMG=quay.io/redhat-cne/cloud-event-consumer:latest

# for HTTP Transport
make deploy-consumer
make undeploy-consumer

# for AMQP Transport
make deploy-consumer-amq
make undeploy-consumer-amq
```
