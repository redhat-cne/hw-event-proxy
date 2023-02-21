# Test Events

The events to be tested are defined in the e2e-tests/data folder with one JSON file per event.

> Naming convention: MessageId-\<notes\>.json

## Temp
- IDRAC.2.8.TMP0110
    - This is the new verison of TMP0110 sent by Dell BMC with IDRAC firmware 6.10.00.00
- TMP0100.json
- TMP0100-no-msg-field.json
    - No message field. Used to test Message Parser.
- TMP0120.json
    - This is a message received from real hardware PowerEdge R640 BIOS=2.8.1, iDRAC Firmware=5.00.00.00
- TMP0120-multiple-records.json
    - A fake event to test multiple EventRecords included in one event
- ZT_Event_Service_Log.1.0.Alert-PSU_1_TEMP_2.json
    - Real message generated on a ZT System (firmware 0.21.0) by lowering `Upper Critical` of sensor `PSU_1_TEMP_2` to below the current temporature.

## Fan failure
- FAN0001.json

## Disk
- STOR1.json

## Power
- PWR1004.json
- iLOEvents.0.9.PowerSupplyRemoved.json (HPE iLO 5)

## Memory
- MEM0004.json
- iLOEvents.2.3.ResourceUpdated.json
  - Real message provided by HPE engineer (HPE iLO 5 firmware 2.60)

## Misc
- RAC1195.json
    - A lifecycle log used to test multiple MessageArgs
