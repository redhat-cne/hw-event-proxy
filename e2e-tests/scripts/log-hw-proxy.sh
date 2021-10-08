#!/bin/bash

NAMESPACE=cloud-native-events

kubectl -n ${NAMESPACE} logs -f -c hw-event-proxy `kubectl -n ${NAMESPACE} get pods | grep hw-event-proxy | cut -f1 -d" "` 

