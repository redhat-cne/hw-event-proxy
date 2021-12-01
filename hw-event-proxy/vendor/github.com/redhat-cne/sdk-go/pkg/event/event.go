// Copyright 2020 The Cloud Native Events Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redhat-cne/sdk-go/pkg/types"
)

// Event represents the canonical representation of a Cloud Native Event.
// Event Json  payload is as follows,
//{
//	"id": "5ce55d17-9234-4fee-a589-d0f10cb32b8e",
//	"type": "event.sync.sync-status.synchronization-state-change",
//	"time": "2021-02-05T17:31:00Z",
//	"data": {
//		"version": "v1.0",
//		"values": [{
//			"resource": "/cluster/node/ptp",
//			"dataType": "notification",
//			"valueType": "enumeration",
//			"value": "ACQUIRING-SYNC"
//			}, {
//			"resource": "/cluster/node/clock",
//			"dataType": "metric",
//			"valueType": "decimal64.3",
//			"value": 100.3
//			}, {
//			"resource": "/cluster/node/temp",
//			"dataType": "notification",
//			"valueType": "redfish-event",
//			"value": {
//			"@odata.context": "/redfish/v1/$metadata#Event.Event",
//			"@odata.type": "#Event.v1_3_0.Event",
//			"Context": "any string is valid",
//			"Events": [{"EventId": "2162", "MemberId": "615703", "MessageId": "TMP0100"}],
//			"Id": "5e004f5a-e3d1-11eb-ae9c-3448edf18a38",
//			"Name": "Event Array"
//			}]
//		}
//}
//Event request model
type Event struct {
	// ID of the event; must be non-empty and unique within the scope of the producer.
	// +required
	ID string `json:"id" example:"789be75d-7ac3-472e-bbbc-6d62878aad4a"`
	// Type - The type of the occurrence which has happened.
	// +required
	Type string `json:"type" example:"event.sync.sync-status.synchronization-state-change"`
	// DataContentType - the Data content type
	// +required
	DataContentType *string `json:"dataContentType" example:"application/json"`
	// Time - A Timestamp when the event happened.
	// +required
	Time *types.Timestamp `json:"time" example:"2021-02-05T17:31:00Z"`
	// DataSchema - A link to the schema that the `Data` attribute adheres to.
	// +optional
	DataSchema *types.URI `json:"dataSchema,omitempty"`
	// +required
	Data *Data `json:"data" `
}

// String returns a pretty-printed representation of the Event.
func (e Event) String() string {
	b := strings.Builder{}
	b.WriteString("  id: " + e.ID + "\r\n")
	b.WriteString("  type: " + e.Type + "\r\n")
	if e.Time != nil {
		b.WriteString("  time: " + e.Time.String() + "\r\n")
	}

	b.WriteString("  data: \r\n")
	b.WriteString("  version: " + e.Data.Version + "\r\n")
	b.WriteString("  values: \r\n")
	for _, v := range e.Data.Values {
		b.WriteString("  value type : " + string(v.ValueType) + "\r\n")
		b.WriteString("  data type : " + string(v.DataType) + "\r\n")
		b.WriteString("  value : " + fmt.Sprintf("%v", v.Value) + "\r\n")
		b.WriteString("  resource: " + v.GetResource() + "\r\n")
	}

	return b.String()
}

// JSONString returns a pretty-printed representation of the Event.
func (e Event) JSONString() string {
	b := strings.Builder{}
	var prettyJSON bytes.Buffer
	eBytes, err := json.Marshal(e)
	if err != nil {
		return e.String()
	}
	err = json.Indent(&prettyJSON, eBytes, "  ", "  ")
	if err != nil {
		return e.String()
	}
	b.Write(prettyJSON.Bytes())

	return b.String()
}

// Clone clones data
func (e Event) Clone() Event {
	out := Event{}
	out.SetData(*e.Data) //nolint:errcheck
	return out
}
