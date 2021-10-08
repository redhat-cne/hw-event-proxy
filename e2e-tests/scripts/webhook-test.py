import json
import time
# disable InsecureRequestWarning: Unverified HTTPS request is being made to host
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

EVENT_TMP0100 = \
{
  "@odata.context": "/redfish/v1/$metadata#Event.Event",
  "@odata.id": "/redfish/v1/EventService/Events/5e004f5a-e3d1-11eb-ae9c-3448edf18a38",
  "@odata.type": "#Event.v1_3_0.Event",
  "Context": "any string is valid",
  "Events": [
    {
      "Context": "any string is valid",
      "EventId": "2162",
      "EventTimestamp": "2021-07-13T15:07:59+0300",
      "EventType": "Alert",
      "MemberId": "615703",
      "Message": "The system board Inlet temperature is less than the lower warning threshold.",
      "MessageArgs": [
        "Inlet"
      ],
      "MessageArgs@odata.count": 1,
      "MessageId": "TMP0100",
      "Severity": "Warning"
    }
  ],
  "Id": "5e004f5a-e3d1-11eb-ae9c-3448edf18a38",
  "Name": "Event Array"
}

encoded_body = json.dumps(EVENT_TMP0100)

http = urllib3.PoolManager(cert_reqs='CERT_NONE')

def send_request():
    http.request('POST', 'https://hw-event-proxy-cloud-native-events.apps.cnfdt15.lab.eng.tlv2.redhat.com/webhook',
                 headers={'Content-Type': 'application/json'},
                 body=encoded_body)

# interval in milliseconds
INTERVAL = 50
while True:
    send_request()
    # time.sleep(INTERVAL/1000)
