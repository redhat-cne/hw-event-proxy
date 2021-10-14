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

test_with_message(){
    MSG_PER_SEC=1
    TEST_DURATION_SEC=10
    INITIAL_DELAY_SEC=10
    CHECK_RESP=YES
    WITH_MESSAGE_FIELD=YES  
}

test_without_message(){
    MSG_PER_SEC=1
    TEST_DURATION_SEC=10
    INITIAL_DELAY_SEC=10
    CHECK_RESP=YES
    WITH_MESSAGE_FIELD=NO  
}

test_performance(){
    MSG_PER_SEC=10
    TEST_DURATION_SEC=600
    INITIAL_DELAY_SEC=30
    CHECK_RESP=YES
    WITH_MESSAGE_FIELD=YES
}

apply_test_options(){
    cat e2e-tests/manifests/redfish-event-test.yaml \
    | sed "/MSG_PER_SEC/{n;s/1/$MSG_PER_SEC/}" \
    | sed "/TEST_DURATION_SEC/{n;s/10/$TEST_DURATION_SEC/}" \
    | sed "/INITIAL_DELAY_SEC/{n;s/10/$INITIAL_DELAY_SEC/}" \
    | sed "/CHECK_RESP/{n;s/YES/$CHECK_RESP/}" \
    | sed "/WITH_MESSAGE_FIELD/{n;s/YES/$WITH_MESSAGE_FIELD/}"
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

# test with message field
echo "--- TEST 1:  WITH MESSAGE FIELD ---"
test_with_message
test

# test without message field
echo "--- TEST 2:  WITHOUT MESSAGE FIELD ---"
test_without_message
test

# performance test
echo "--- TEST 3:  PERFORMANCE TEST ---"
test_performance
test


test() {
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
            head -10 ${LOG_DIR}/_report.csv
            echo "*** TEST PASSED ***"
            echo "--- Delete test pod ---"
            kubectl delete -f e2e-tests/manifests/redfish-event-test.yaml 2>/dev/null || true
        else
            echo "*** TEST FAILED ***: Events sent: $num_events_send, Events received: num_events_received"
            # do not delete the test pod in case it's needed for debug
            exit 1
    fi
}