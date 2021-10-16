# End to End Tests for Redfish Events


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

```

## Undeply
```
oc delete job redfish-event-test
```
