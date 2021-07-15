import logging

import sushy
import json
from sushy import auth
from sushy.resources import base
from sushy.resources.registry import message_registry

# disable InsecureRequestWarning: Unverified HTTPS request is being made to host
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# Enable logging at DEBUG level
LOG = logging.getLogger('sushy')
LOG.setLevel(logging.DEBUG)
LOG.addHandler(logging.StreamHandler())

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

basic_auth = auth.BasicAuth(username='root', password='calvin')

s = sushy.Sushy('https://10.46.61.142/redfish/v1',
                auth=basic_auth, verify=False)

# Get the Redfish version
print(s.redfish_version)
registries = s.lazy_registries

# preload the registries
registries.registries

m = base.MessageListField(EVENT_TMP0100["Events"][0])

message_field = base.MessageListField('Foo')
message_field.message_id = 'TMP0100'
message_field.message_args = ['Inlet']
message_field.severity = None
message_field.resolution = None

print("\nOriginal:\n")
print(message_field.message_id)
print(message_field.message)

message_registry.parse_message(registries, message_field)
print("\nParsed:\n")
print(message_field.message_id)
print(message_field.message)
print(message_field.severity)
print(message_field.resolution)
