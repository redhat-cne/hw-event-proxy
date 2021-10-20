#!/usr/bin/env python3

# verify event message contents
import re
import sys

LOG_DIR = './logs'

# example event receveid:
# time="2021-10-20T19:30:49Z" level=debug msg="received event {id:f1476c3a-5aff-41ec-9a92-bcd5d3d7cf5f,type:HW_EVENT,dataContentType:application/json,time:2021-10-20T19:30:49.088Z,data:{version:v1,data:{@odata.context:/redfish/v1/$metadata#Event.Event,Context:any string is valid,@odata.type:#Event.v1_3_0.Event,Events:[{Context:any string is valid,EventGroupId:0,EventId:2162,EventTimestamp:2021-07-13T15:07:59+0300,Message:The system board Inlet temperature is less than the lower warning threshold.,MessageArgs:[Inlet],Severity:Warning,EventType:Alert,MessageId:TMP0100,MemberId:615703}],Id:5e004f5a-e3d1-11eb-ae9c-3448edf18a38,Name:Event Array}}}"

MATCHES = dict()
MATCHES['EventGroupId'] = '0'
MATCHES['EventId'] = '2162'
MATCHES['EventTimestamp'] = '2021-09-14T02:28:55+0300'
MATCHES['EventType'] = 'Alert'
MATCHES['MemberId'] = '679424'
MATCHES['Message'] = 'The system inlet temperature is greater than the upper warning threshold.'
MATCHES['MessageArgs'] = '[Inlet]'
MATCHES['MessageId'] = 'TMP0120'
MATCHES['OriginOfCondition'] = '{"@odata.id":"/redfish/v1/Systems/System.Embedded.1"}'
MATCHES['Severity'] = 'Warning'
MATCHES['Id'] = 'eef31690-f615-11eb-95ea-3448edf18a38'

def main():
    log_dir = sys.argv[1] if len(sys.argv) > 1 else LOG_DIR
    event_received = None
    with open(log_dir + '/event-received.log', 'r') as reader:
        event_received = reader.readline()
    print(event_received)

    for k, v in MATCHES.items():
        if compare(event_received, k, v) == -1:
            sys.exit(1)

def compare(received, k, expected):
    pattern = ',{}:(.*?)[,}}]'.format(k)
    print("pattern: {}".format(pattern))
    m = re.search(pattern, received)
    if not m:
        print("Error: pattern {} not found".format(pattern))
        return -1
    actual = m.group(1)
    if actual!= expected:
        print("Error: expected: {}, actual: {}".format(expected, actual))
        return -1
    return 0

if __name__ == '__main__':
    main()