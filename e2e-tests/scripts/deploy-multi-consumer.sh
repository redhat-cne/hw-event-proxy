#!/bin/bash
# run this script at root dir of the repo
set -e

# action is `deploy` or `undeploy`
ACTION=$1
# number of additional consumers need to deploy/undeploy
NUM_OF_CONSUMER=`expr $2 - 1`

NAMESPACE=openshift-bare-metal-events
CONSUMER_ROOT=manifests/consumer

if [ "$ACTION" == "deploy" ]; then
  ACTION="apply"
  echo "---deploying $NUM_OF_CONSUMER more consumers---"
else
  ACTION="delete"
fi

WORK_DIR=`mktemp -d`

# check if tmp dir was created
if [[ ! "$WORK_DIR" || ! -d "$WORK_DIR" ]]; then
  echo "Could not create temp dir"
  exit 1
fi

# deletes the temp directory
function cleanup {      
  rm -rf "$WORK_DIR"
  echo "Deleted temp working directory $WORK_DIR"
}

# register the cleanup function to be called on the EXIT signal
trap cleanup EXIT

for i in `seq $NUM_OF_CONSUMER`
do
  cp ${CONSUMER_ROOT}/* ${WORK_DIR}/
  sed -i "s/ consumer/ consumer-$i/g" ${WORK_DIR}/service.yaml
  sed -i "s/consumer-sidecar-service/consumer-$i-sidecar-service/g" ${WORK_DIR}/service.yaml
  sed -i "s/sidecar-consumer-secret/sidecar-consumer-$i-secret/g" ${WORK_DIR}/service.yaml
  sed -i "s/ consumer/ consumer-$i/g" ${WORK_DIR}/deployment.yaml
  sed -i "s/consumer-events-subscription-service/consumer-$i-events-subscription-service/g" ${WORK_DIR}/deployment.yaml
  sed -i "s/sidecar-consumer-secret/sidecar-consumer-$i-secret/g" ${WORK_DIR}/deployment.yaml
  # remove the unchanged files
  sed -i '/roles.yaml/d' ${WORK_DIR}/kustomization.yaml
  sed -i '/service-account.yaml/d' ${WORK_DIR}/kustomization.yaml
  kustomize build ${WORK_DIR} | kubectl $ACTION -f -
done
