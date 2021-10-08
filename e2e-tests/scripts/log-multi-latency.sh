#!/bin/bash

# generate logs for multiple consumers.

NAMESPACE=cloud-native-events
LOG_DIR=/home/jacding/logs

for c in `kubectl -n ${NAMESPACE} get pods | grep consumer| cut -f1 -d" "`; do
    kubectl -n ${NAMESPACE} logs -f -c cloud-native-event-consumer $c >> ${LOG_DIR}/_latency_$c.log &
done

kubectl -n ${NAMESPACE} logs -f `kubectl -n ${NAMESPACE} get pods | grep redfish-event-test | cut -f1 -d" "` >> ${LOG_DIR}/_test_tool.log &

