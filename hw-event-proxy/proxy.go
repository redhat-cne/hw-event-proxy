// Copyright 2021 The Cloud Native Events Authors
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

package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	hwevent "github.com/redhat-cne/sdk-go/pkg/hwevent"

	"github.com/redhat-cne/sdk-go/pkg/channel"
	"github.com/redhat-cne/sdk-go/pkg/pubsub"
	"github.com/redhat-cne/sdk-go/pkg/types"

	"github.com/redhat-cne/hw-event-proxy/hw-event-proxy/pb"
	"github.com/redhat-cne/hw-event-proxy/hw-event-proxy/restclient"
	"google.golang.org/grpc"

	v1amqp "github.com/redhat-cne/sdk-go/v1/amqp"
	v1hwevent "github.com/redhat-cne/sdk-go/v1/hwevent"
	v1pubsub "github.com/redhat-cne/sdk-go/v1/pubsub"
	log "github.com/sirupsen/logrus"
)

// SCConfiguration simple configuration to initialize variables
type SCConfiguration struct {
	EventInCh  chan *channel.DataChan
	EventOutCh chan *channel.DataChan
	CloseCh    chan struct{}
	APIPort    int
	APIPath    string
	PubSubAPI  *v1pubsub.API
	StorePath  string
	AMQPHost   string
	BaseURL    *types.URI
}

var (
	resourceAddress string = "/hw-event"
	// used by the webhook handlers
	scConfig  *SCConfiguration
	pub       pubsub.PubSub
	eventType string = "HW_EVENT"
)

// Start hw event plugin to process events,metrics and status, expects rest api available to create publisher and subscriptions
func Start(wg *sync.WaitGroup, config *SCConfiguration, fn func(e interface{}) error) error { //nolint:deadcode,unused
	scConfig = config

	// create publisher
	var err error

	returnURL := fmt.Sprintf("%s%s", config.BaseURL, "dummy")
	//	pub, err = scConfig.PubSubAPI.CreatePublisher(v1pubsub.NewPubSub(scConfig.BaseURL, resourceAddress))
	pub, err = scConfig.PubSubAPI.CreatePublisher(v1pubsub.NewPubSub(types.ParseURI(returnURL), resourceAddress))
	if err != nil {
		log.Errorf("failed to create a publisher %v", err)
		return err
	}
	log.Infof("Created publisher %v", pub)

	// once the publisher response is received, create a transport sender object to send events.
	v1amqp.CreateSender(scConfig.EventInCh, pub.GetResource())
	log.Infof("Created sender %v", pub.GetResource())

	startWebhook()

	return nil
}

func startWebhook() {
	http.HandleFunc("/ack/event", ackEvent)
	http.HandleFunc("/webhook", publishEvent)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", GetIntEnv("HW_EVENT_PORT")), nil)
		if err != nil {
			log.Errorf("error with webhook server %s\n", err.Error())
		}
	}()
}

func ackEvent(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("error reading acknowledgment %v", err)
	}
	e := string(bodyBytes)
	if e != "" {
		log.Debugf("received ack %s", string(bodyBytes))
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// publishEvent gets redfish HW events and converts it to cloud native event
// and publishes to the event framework publisher
func publishEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("error reading hw event: %v", err)
		return
	}
	event := createHwEvent()
	redfishEvent := hwevent.RedfishEvent{}
	err = json.Unmarshal(bodyBytes, &redfishEvent)
	if err != nil {
		log.Errorf("failed to unmarshal hw event: %v", err)
		return
	}
	for i, e := range redfishEvent.Events {
		if e.Message == "" {
			parsed, err := parseMessage(e)
			if err == nil {
				redfishEvent.Events[i] = parsed
			} else {
				// ignore error
				log.Debugf("error parsing message: %v", err)
			}

		}
	}

	data := v1hwevent.CloudNativeData()
	data.SetVersion("v1") //nolint:errcheck
	data.SetData(&redfishEvent)
	event.SetData(data)
	_ = publish(scConfig, event)
}

func parseMessage(m hwevent.EventRecord) (hwevent.EventRecord, error) {
	addr := "localhost:9999"
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return hwevent.EventRecord{}, err
	}
	defer conn.Close()

	client := pb.NewMessageParserClient(conn)
	req := &pb.ParserRequest{
		MessageId:   m.MessageID,
		MessageArgs: m.MessageArgs,
	}

	resp, err := client.Parse(context.Background(), req)
	if err != nil {
		return hwevent.EventRecord{}, err
	}

	m.Message = resp.Message
	m.Severity = resp.Severity
	m.Resolution = resp.Resolution
	return m, nil
}

func createHwEvent() hwevent.Event {
	event := v1hwevent.CloudNativeEvent()
	event.ID = pub.ID
	event.Type = eventType
	event.SetTime(types.Timestamp{Time: time.Now().UTC()}.Time)
	event.SetDataContentType(hwevent.ApplicationJSON)
	return event
}

func publish(scConfig *SCConfiguration, e hwevent.Event) error {
	//create publisher
	url := fmt.Sprintf("%s%s", scConfig.BaseURL.String(), "create/hwevent")
	rc := restclient.New()
	b, err := json.Marshal(e)
	if err != nil {
		log.Errorf("error marshalling event %v", e)
		return err
	}
	if status := rc.Post(types.ParseURI(url), b); status == http.StatusBadRequest {
		return fmt.Errorf("post returned status %d", status)
	}
	log.Debugf("published hw event %s", e)
	return nil
}
