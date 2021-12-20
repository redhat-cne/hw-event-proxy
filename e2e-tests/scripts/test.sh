#!/bin/bash

COLOR_RESET='\033[0m'
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW="\033[1;33m"
BOLD='\033[1m'

NAMESPACE=cloud-native-events
LOG_DIR=./logs
DATA_DIR=e2e-tests/data
job_result=0
perf=0
verbose=0

# Performance target for Intra-Node:
# At a rate of 10 msgs/sec, 95% of the massages should have latency <= 10ms.
# Should support this performance with multiple (10~20) recipients.
PERF_TARGET_PERCENT_10MS=95

Help()
{
   echo "$0 [-p|-h]"
   echo "options:"
   echo "-p  Performance tests."
   echo "-h  Print this Help."
   echo
}

while getopts ":hpv" option; do
   case $option in
      h) Help
         exit;;
      p) perf=1;;
     \?) echo "Error: Invalid option"
         exit;;
   esac
done

wait_for_resource(){
    resoure_name=$1
    condition=$2
    timeout=$3
    while true; do
        if kubectl -n ${NAMESPACE} wait --for=condition=$condition --timeout=$timeout $resoure_name  >/dev/null 2>&1; then
            job_result=0
            break
        fi
        if kubectl -n ${NAMESPACE} wait --for=condition=failed --timeout=$timeout $resoure_name  >/dev/null 2>&1; then
            job_result=1
            break
        fi
        sleep 3
    done
}

check_images(){
    TIMEOUT_SEC=120
    start_time="$(date -u +%s)"
    while true; do
        current_time="$(date -u +%s)"
        elapsed_seconds=$(($current_time-$start_time))
        if [ $elapsed_seconds -gt $TIMEOUT_SEC ]; then
            echo "timeout of $TIMEOUT_SEC sec"
            break
        fi
        kubectl -n ${NAMESPACE} get pods || true
        kubectl -n ${NAMESPACE} get pod `kubectl -n ${NAMESPACE} get pods | grep hw-event-proxy | cut -f1 -d" "` -o json | jq .status.containerStatuses || true
        sleep 1
    done
}



cleanup_logs(){
    rm -f ${LOG_DIR}/* 2>/dev/null
}

cleanup_logs_pid(){
    for pidFile in ${LOG_DIR}/*.pid; do
        if test -f "$pidFile"; then
            pkill -F $pidFile 2>/dev/null
            rm -f $pidFile 2>/dev/null
        fi
    done
}

cleanup_test_pod(){
    kubectl -n ${NAMESPACE} delete job/redfish-event-test --ignore-not-found=true --grace-period=0 >/dev/null 2>&1 || true
    kubectl -n ${NAMESPACE} wait --for=delete job/redfish-event-test --timeout=60s 2>/dev/null || true
}

fail_test(){
    cleanup_logs_pid
    echo "--- hw-event-proxy logs ---"
    hw_event_proxy_pod=`kubectl -n ${NAMESPACE} get pods | grep hw-event-proxy | cut -f1 -d" "`
    kubectl -n ${NAMESPACE} logs --tail=50 -c hw-event-proxy $hw_event_proxy_pod >> ${LOG_DIR}/last_log_$hw_event_proxy_pod.log &
    for consumer_pod in `kubectl -n ${NAMESPACE} get pods | grep consumer| cut -f1 -d" "`; do
         echo "--- consumer $consumer_pod logs ---"
         kubectl -n ${NAMESPACE} logs --tail=50 -c cloud-native-event-consumer $consumer_pod >> ${LOG_DIR}/last_log_$consumer_pod.log &
    done

    echo "Check directory ${LOG_DIR} for more logs."
    echo -e "***$RED TEST FAILED $COLOR_RESET***"
    exit 1
}


test_sanity() {
    # streaming logs for multiple consumers.
    echo "--- Start streaming consumer logs ---"
    consumer_pod=`kubectl -n ${NAMESPACE} get pods | grep consumer| cut -f1 -d" "`
    kubectl -n ${NAMESPACE} logs -f --tail=1 -c cloud-native-event-consumer $consumer_pod >> ${LOG_DIR}/$consumer_pod.log &
    echo "$!" > ${LOG_DIR}/log-$consumer_pod.pid

    # start the test
    echo "--- Start testing ---"
    kubectl -n ${NAMESPACE} apply -f e2e-tests/manifests/redfish-event-test.yaml

    # streaming logs for the test tool
    kubectl -n ${NAMESPACE} wait --for=condition=ready pod -l app=redfish-event-test --timeout=60s  >/dev/null 2>&1
    kubectl -n ${NAMESPACE} logs -f `kubectl -n ${NAMESPACE} get pods | grep redfish-event-test | cut -f1 -d" "` >> ${LOG_DIR}/redfish-event-test.log &
    echo "$!" > ${LOG_DIR}/log-redfish-event-test.pid

    wait_for_resource job/redfish-event-test complete 0 >/dev/null
    if [[ $job_result -eq 1 ]]; then
        fail_test
    fi

    echo "--- Check test result ---"
    grep "received event" ${LOG_DIR}/$consumer_pod.log | sed 's/\\\"//g' >> ${LOG_DIR}/event-received.log

    for eventFile in ${DATA_DIR}/*.json; do
        e2e-tests/scripts/verify-sanity.py $eventFile ${LOG_DIR}/event-received.log
        if [[ $? -eq 1 ]]; then
            fail_test
        fi
    done

    echo -e "***$GREEN TEST PASSED 😄 $COLOR_RESET***"
}

test_perf() {

    # streaming logs for multiple consumers.
    echo "--- Start streaming consumer logs ---"
    for consumer_pod in `kubectl -n ${NAMESPACE} get pods | grep consumer| cut -f1 -d" "`; do
        kubectl -n ${NAMESPACE} logs -f -c cloud-native-event-consumer $consumer_pod | grep "Latency for hardware event" >> ${LOG_DIR}/$consumer_pod.log &
        echo "$!" > ${LOG_DIR}/log-$consumer_pod.pid
    done

    MSG_PER_SEC=10
    TEST_DURATION_SEC=600
    INITIAL_DELAY_SEC=60

    cat e2e-tests/manifests/redfish-event-test.yaml \
    | sed "/PERF/{n;s/NO/YES/}" \
    | sed "/MSG_PER_SEC/{n;s/1/$MSG_PER_SEC/}" \
    | sed "/TEST_DURATION_SEC/{n;s/10/$TEST_DURATION_SEC/}" \
    | sed "/INITIAL_DELAY_SEC/{n;s/10/$INITIAL_DELAY_SEC/}" > ${LOG_DIR}/redfish-event-test.yaml

    # start the test
    echo "--- Start testing ---"
    kubectl -n ${NAMESPACE} apply -f ${LOG_DIR}/redfish-event-test.yaml

    # streaming logs for the test tool
    kubectl -n ${NAMESPACE} wait --for=condition=ready pod -l app=redfish-event-test --timeout=60s  >/dev/null 2>&1
    kubectl -n ${NAMESPACE} logs -f `kubectl -n ${NAMESPACE} get pods | grep redfish-event-test | cut -f1 -d" "` >> ${LOG_DIR}/redfish-event-test.log &
    echo "$!" > ${LOG_DIR}/log-redfish-event-test.pid

    if [[ $perf -eq 1 ]]; then
        echo "Test will run for $(( ($TEST_DURATION_SEC + $INITIAL_DELAY_SEC)/60 )) minutes."
    fi

    wait_for_resource job/redfish-event-test complete 0  >/dev/null 2>&1
    if [[ $job_result -eq 1 ]]; then
        fail_test
    fi

    echo "Sleep for 5 seconds: wait for logs to complete streaming"
    sleep 5

    echo "--- Check test result ---"
    e2e-tests/scripts/verify-perf.py
    if [[ $? -eq 1 ]]; then
        fail_test
    fi

    num_events_send=$(grep 'Total Msg Sent:' ${LOG_DIR}/redfish-event-test.log | cut -f6 -d" " | sed 's/"$//')
    num_events_received=$(grep -rIn "Events per Consumer" ${LOG_DIR}/_report.csv | sed 's/.*\t//')
    head -10 ${LOG_DIR}/_report.csv
    percent_10ms=$(grep 'Percentage within 10ms' ${LOG_DIR}/_report.csv | sed 's/.*\t//' | sed 's/\..*//')
    echo
    echo "Performance target: 95% of the massages have latency within 10ms."
    if [ $percent_10ms -lt $PERF_TARGET_PERCENT_10MS ]; then
        echo -e "$RED Error: Performance actual: ${percent_10ms}% of the massages have latency within 10ms. $COLOR_RESET"
        fail_test
    fi
    echo -e "***$GREEN TEST PASSED $COLOR_RESET***"
    echo
    echo "Full test report is available at ${LOG_DIR}/_report.csv"
}

mkdir -p -- "$LOG_DIR"
echo "--- Cleanup previous test pod and logs---"
cleanup_test_pod
cleanup_logs_pid
cleanup_logs

echo "--- Check if consumer pod is available ---"
wait_for_resource deployment/consumer available 60s >/dev/null 2>&1
if [[ $job_result -eq 1 ]]; then
    echo "Consumer pod is not available"
    exit 1
fi

check_images

echo "--- Check if hw-event-proxy pod is available ---"
wait_for_resource deployment/hw-event-proxy available 60s >/dev/null 2>&1
if [[ $job_result -eq 1 ]]; then
    echo "hw-event-proxy pod is not available"
    exit 1
fi

# This help verifying the image used is the latest image built from CI
echo "Hw-event-proxy container status:"
kubectl -n ${NAMESPACE} get pod `kubectl -n ${NAMESPACE} get pods | grep hw-event-proxy | cut -f1 -d" "` -o json | jq .status.containerStatuses

if [[ $perf -eq 1 ]]; then
    echo -e "---$BOLD PERFORMANCE TEST $COLOR_RESET---"
    test_perf
else
    echo -e "---$BOLD SANITY TEST $COLOR_RESET---"
    test_sanity
fi

cleanup_logs_pid
