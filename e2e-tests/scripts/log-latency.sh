#!/bin/bash

NAMESPACE=cloud-native-events
LATENCY_LOG=/home/jacding/logs/_latency.log

kubectl -n ${NAMESPACE} logs -f -c cloud-native-event-consumer `kubectl -n ${NAMESPACE} get pods | grep cloud-native-consumer-deployment | cut -f1 -d" "` >> ${LATENCY_LOG} &

kubectl -n ${NAMESPACE} logs -f `kubectl -n ${NAMESPACE} get pods | grep redfish-event-test | cut -f1 -d" "` >> ${LATENCY_LOG} &

