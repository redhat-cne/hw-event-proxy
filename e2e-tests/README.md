# Performance Tests for Redfish Events


## Build Image
```
make build
scripts/build-image.sh
podman images
TAG=xxx
podman push localhost/redfish-event-test:${TAG} quay.io/redhat_emp1/redfish-event-test:latest
```

## Run Tests
```
oc apply -f manifests/redfish-event-test.yaml
oc logs -f cloud-native-consumer-deployment-cdb95bfd8-h68xs cloud-native-event-consumer | grep "Latency for hardware event" >> ~/logs/_latency.log
```

## Undeply
```
oc delete job redfish-event-test
```
