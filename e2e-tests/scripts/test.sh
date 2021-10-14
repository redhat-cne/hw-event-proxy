#!/bin/bash

NAMESPACE=cloud-native-events
LOG_DIR=./logs
job_result=0

wait_for_resource(){
    resoure_name=$1
    condition=$2
    timeout=$3
    while true; do
        if kubectl wait --for=condition=$condition --timeout=$timeout $resoure_name 2>/dev/null; then
            job_result=0
            break
    fi

    if kubectl wait --for=condition=failed --timeout=$timeout $resoure_name 2>/dev/null; then
        job_result=1
        break
    fi

    sleep 3
    done
}

# clean up logs
echo "--- Remove previous logs ---"
mkdir -p -- "$LOG_DIR"
rm -f $LOG_DIR/*.log
rm -f $LOG_DIR/*.csv

echo "--- Check if consumer pod is available ---"
wait_for_resource deployment/consumer available 60s
if [[ $job_result -eq 1 ]]; then
    echo "Consumer pod is not available"
    exit 1
fi

echo "--- Check if hw-event-proxy pod is available ---"
wait_for_resource deployment/hw-event-proxy available 60s
if [[ $job_result -eq 1 ]]; then
    echo "hw-event-proxy pod is not available"
    exit 1
fi

# streaming logs for multiple consumers.
echo "--- Start streaming consumer logs ---"
for podname in `kubectl -n ${NAMESPACE} get pods | grep consumer| cut -f1 -d" "`; do
    kubectl -n ${NAMESPACE} logs -f -c cloud-native-event-consumer $podname >> ${LOG_DIR}/$podname.log &
done

# start the test
echo "--- Start testing ---"
kubectl apply -f e2e-tests/manifests/redfish-event-test.yaml

wait_for_resource job/redfish-event-test complete 0
if [[ $job_result -eq 1 ]]; then
    echo "redfish-event-test job is not complete"
    exit 1
fi

echo "--- Test completed. Collecting test tool logs ---"
# streaming logs for the test tool
kubectl -n ${NAMESPACE} logs -f `kubectl -n ${NAMESPACE} get pods | grep redfish-event-test | cut -f1 -d" "` >> ${LOG_DIR}/redfish-event-test.log &

echo "--- Generate test report ---"
e2e-tests/scripts/parse-multi-logs.py

echo "--- Check test result ---"
num_events_send=$(grep 'Total Msg sent:' ${LOG_DIR}/redfish-event-test.log | cut -f6 -d" " | sed 's/"$//')
num_events_received=$(grep -rIn "Total Events" ${LOG_DIR}/_report.csv | sed 's/.*\t//')
if [ $num_events_send -eq $num_events_received ]
    then
        echo "TEST PASSED"
    else
        echo "TEST FAILED: Events sent: $num_events_send, Events received: num_events_received"
fi