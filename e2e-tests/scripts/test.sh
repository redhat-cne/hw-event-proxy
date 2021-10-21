#!/bin/bash

COLOR_RESET='\033[0m'
GREEN='\033[1;32m'
RED='\033[1;31m'
BOLD='\033[1m'

NAMESPACE=cloud-native-events
LOG_DIR=./logs
job_result=0
perf=0
verbose=0
num_of_consumer=0

# Performance target for Intra-Node:
# At a rate of 10 msgs/sec, 95% of the massages should have latency <= 10ms.
# Should support this performance with multiple (10~20) recipients.
PERF_TARGET_PERCENT_10MS=95

Help()
{
   echo "$0 [-p|-h|-v]"
   echo "options:"
   echo "-p  Performance tests."
   echo "-v  Verbose mode."
   echo "-h  Print this Help."
   echo
}

while getopts ":hpv" option; do
   case $option in
      h) Help
         exit;;
      p) perf=1;;
      v) verbose=1;;
     \?) echo "Error: Invalid option"
         exit;;
   esac
done

wait_for_resource(){
    resoure_name=$1
    condition=$2
    timeout=$3
    while true; do
        if kubectl -n ${NAMESPACE} wait --for=condition=$condition --timeout=$timeout $resoure_name 2>/dev/null; then
            job_result=0
            break
    fi

    if kubectl -n ${NAMESPACE} wait --for=condition=failed --timeout=$timeout $resoure_name 2>/dev/null; then
        job_result=1
        break
    fi

    sleep 3
    done
}

apply_test_options(){
    cat e2e-tests/manifests/redfish-event-test.yaml \
    | sed "/MSG_PER_SEC/{n;s/1/$MSG_PER_SEC/}" \
    | sed "/TEST_DURATION_SEC/{n;s/10/$TEST_DURATION_SEC/}" \
    | sed "/INITIAL_DELAY_SEC/{n;s/10/$INITIAL_DELAY_SEC/}" \
    | sed "/CHECK_RESP/{n;s/YES/$CHECK_RESP/}" \
    | sed "/WITH_MESSAGE_FIELD/{n;s/YES/$WITH_MESSAGE_FIELD/}" > ${LOG_DIR}/redfish-event-test.yaml
}

test_with_message(){
    MSG_PER_SEC=1
    TEST_DURATION_SEC=10
    INITIAL_DELAY_SEC=2
    CHECK_RESP=YES
    WITH_MESSAGE_FIELD=YES  
    apply_test_options
}

test_without_message(){
    MSG_PER_SEC=1
    TEST_DURATION_SEC=10
    # wait a random duration between 1 to 60 seconds for preloading Redfish Registries
    INITIAL_DELAY_SEC=$(( $RANDOM % 60 + 1 ))
    CHECK_RESP=YES
    WITH_MESSAGE_FIELD=NO  
    apply_test_options
}

test_performance(){
    MSG_PER_SEC=10
    TEST_DURATION_SEC=600
    INITIAL_DELAY_SEC=60
    CHECK_RESP=YES
    WITH_MESSAGE_FIELD=YES
    apply_test_options
}

reset_logs(){
    # empty log files without breaking the streaming
    truncate -s 0 ${LOG_DIR}/consumer*.log 2>/dev/null
    truncate -s 0 ${LOG_DIR}/redfish-event-test.log 2>/dev/null
}

cleanup_logs(){
    rm -f ${LOG_DIR}/consumer*.log 2>/dev/null
    rm -f ${LOG_DIR}/redfish-event-test.log 2>/dev/null
    rm -f ${LOG_DIR}/_report.csv 2>/dev/null
    rm -f ${LOG_DIR}/redfish-event-test.yaml 2>/dev/null
}

cleanup_consumer_logs(){
    rm -f ${LOG_DIR}/consumer*.log 2>/dev/null
}

cleanup_log_streaming(){
    for pidFile in ${LOG_DIR}/*.pid; do
        if test -f "$pidFile"; then
            pkill -F $pidFile 2>/dev/null
            rm -f $pidFile 2>/dev/null
        fi
    done
}

cleanup_log_streaming_test(){
    pidFile=${LOG_DIR}/log-redfish-event-test.pid
    if test -f "$pidFile"; then
        pkill -F $pidFile 2>/dev/null
        rm -f $pidFile 2>/dev/null
    fi
}

debug_log(){
    if [[ $verbose -eq 1 ]]; then
        echo $1
    fi
}

cleanup_test_pod(){
    kubectl -n ${NAMESPACE} delete job/redfish-event-test --ignore-not-found=true --grace-period=0 >/dev/null 2>&1 || true
    kubectl -n ${NAMESPACE} wait --for=delete job/redfish-event-test --timeout=60s 2>/dev/null || true
    cleanup_log_streaming_test
}

run_test() {
    debug_log "--- Cleanup previous test pod and logs---"
    cleanup_test_pod
    reset_logs

    # start the test
    debug_log "--- Start testing ---"
    kubectl -n ${NAMESPACE} apply -f ${LOG_DIR}/redfish-event-test.yaml >/dev/null

    # streaming logs for the test tool
    kubectl -n ${NAMESPACE} wait --for=condition=ready pod -l app=redfish-event-test --timeout=60s  >/dev/null 2>&1
    kubectl -n ${NAMESPACE} logs -f `kubectl -n ${NAMESPACE} get pods | grep redfish-event-test | cut -f1 -d" "` >> ${LOG_DIR}/redfish-event-test.log &
    echo "$!" > ${LOG_DIR}/log-redfish-event-test.pid

    if [[ $perf -eq 1 ]]; then
        echo "Test will run for $(( ($TEST_DURATION_SEC + $INITIAL_DELAY_SEC)/60 )) minutes."
    fi
    wait_for_resource job/redfish-event-test complete 0 >/dev/null
    if [[ $job_result -eq 1 ]]; then
        echo "redfish-event-test job is not complete"
        cleanup_log_streaming
        exit 1
    fi

    debug_log "Sleep for 5 seconds: wait for logs to complete streaming"
    sleep 5
    debug_log "--- Generate test report ---"
    e2e-tests/scripts/parse-logs.py

    debug_log "--- Check test result ---"
    num_events_send=$(grep 'Total Msg Sent:' ${LOG_DIR}/redfish-event-test.log | cut -f6 -d" " | sed 's/"$//')
    num_events_received=$(grep -rIn "Events per Consumer" ${LOG_DIR}/_report.csv | sed 's/.*\t//')
    if [ $num_events_send -eq $num_events_received ]; then
        head -10 ${LOG_DIR}/_report.csv
        if [[ $perf -eq 1 ]]; then
            percent_10ms=$(grep 'Percentage within 10ms' ${LOG_DIR}/_report.csv | sed 's/.*\t//' | sed 's/\..*//')
            if [ $percent_10ms -lt $PERF_TARGET_PERCENT_10MS ]; then
                echo -e "***$RED TEST FAILED $COLOR_RESET***"
                echo "Performance target: 95% of the massages have latency <= 10ms."
                echo "Performance actual: ${percent_10ms}% of the massages have latency <= 10ms."
                cleanup_log_streaming
                exit 1
            fi
        fi
        echo -e "***$GREEN TEST PASSED $COLOR_RESET***"
        echo
    else
        echo -e "***$RED TEST FAILED $COLOR_RESET***: Events sent: $num_events_send, Events received: $num_events_received"
        # do not delete the test pod in case it's needed for debug
        cleanup_log_streaming
        exit 1
    fi
}

mkdir -p -- "$LOG_DIR"
debug_log "--- Clean up logs ---"
cleanup_logs
cleanup_log_streaming

debug_log "--- Check if consumer pod is available ---"
wait_for_resource deployment/consumer available 60s >/dev/null
if [[ $job_result -eq 1 ]]; then
    echo "Consumer pod is not available"
    exit 1
fi

debug_log "--- Check if hw-event-proxy pod is available ---"
wait_for_resource deployment/hw-event-proxy available 60s >/dev/null
if [[ $job_result -eq 1 ]]; then
    echo "hw-event-proxy pod is not available"
    exit 1
fi

# streaming logs for multiple consumers.
debug_log "--- Start streaming consumer logs ---"
for podname in `kubectl -n ${NAMESPACE} get pods | grep consumer| cut -f1 -d" "`; do
    kubectl -n ${NAMESPACE} logs -f -c cloud-native-event-consumer $podname >> ${LOG_DIR}/$podname.log &
    echo "$!" > ${LOG_DIR}/log-$podname.pid
    num_of_consumer=$(( num_of_consumer + 1 ))
done

if [[ $perf -eq 0 ]]; then
    # test with message field
    echo -e "---$BOLD TEST 1:  WITH MESSAGE FIELD $COLOR_RESET---"
    test_with_message
    run_test

    # test without message field
    echo -e "---$BOLD TEST 2:  WITHOUT MESSAGE FIELD $COLOR_RESET---"
    test_without_message
    echo "Wait $INITIAL_DELAY_SEC seconds for preloading Redfish Registries..."
    run_test
else
    # performance test
    echo -e "---$BOLD PERFORMANCE TEST $COLOR_RESET---"
    test_performance
    run_test
fi

cleanup_log_streaming
cleanup_consumer_logs
echo "Full test report is available at ${LOG_DIR}/_report.csv"
