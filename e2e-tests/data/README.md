# Test Events

The events to be tested are defined in the e2e-tests/data folder with one JSON file per event.

> Naming convention: MessageId-<notes>.json

## Temp
- TMP0100.json
- TMP0100-no-msg-field.json
    - No message field. Used to test Message Parser.
- TMP0120.json
    - This is a message received from real hardware PowerEdge R640 BIOS=2.8.1, iDRAC Firmware=5.00.00.00

## Fan failure
- FAN0001.json

## Disk
- STOR1.json

## Power
- PWR1004.json

## Memory
- MEM0004.json

## Misc
- RAC1195.json
    - A lifecycle log used to test multiple MessageArgs

