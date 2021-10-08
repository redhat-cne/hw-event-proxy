#!/bin/bash

NUM_CONSUMER=${1:-1}
DEPLOYMENT_NAME=cloud-native-consumer-deployment
PATH_YAML=/home/jacding/repo/jzding/playground/redfish-event-test/manifests
i=1; while [ $i -le ${NUM_CONSUMER} ]; do
    sed "s/$DEPLOYMENT_NAME/consumer-$i/g" $PATH_YAML/consumer.yaml | oc apply -f -
    i=$(($i + 1))
done