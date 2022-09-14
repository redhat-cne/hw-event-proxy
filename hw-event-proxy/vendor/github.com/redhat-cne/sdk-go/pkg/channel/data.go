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

package channel

import (
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

// DataChan ...
type DataChan struct {
	ID       string
	ClientID uuid.UUID
	Address  string
	Data     *cloudevents.Event
	Status   Status
	//Type defines type of data (Notification,Metric,Status)
	Type Type
	// OnReceiveFn  to do on OnReceive
	OnReceiveFn func(e cloudevents.Event)
	// OnReceiveOverrideFn Optional for event, but override for status pings.This is an override function on receiving msg by amqp listener,
	// if not set then the data is sent to out channel and processed by side-car  default method
	OnReceiveOverrideFn func(e cloudevents.Event, dataChan *DataChan) error
	// ProcessEventFn  Optional, this allows to customize message handler thar was received at the out channel
	ProcessEventFn func(e interface{}) error
}

// CreateCloudEvents ...
func (d *DataChan) CreateCloudEvents(dataType string) (*cloudevents.Event, error) {
	ce := cloudevents.NewEvent(cloudevents.VersionV03)
	ce.SetDataContentType(cloudevents.ApplicationJSON)
	ce.SetSpecVersion(cloudevents.VersionV03)
	ce.SetType(dataType)
	ce.SetSource(d.Address)
	ce.SetID(d.ClientID.String())
	return &ce, nil
}
