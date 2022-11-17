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

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/redhat-cne/sdk-go/pkg/event"
	"github.com/redhat-cne/sdk-go/pkg/event/redfish"
	"github.com/redhat-cne/sdk-go/pkg/pubsub"
	"github.com/redhat-cne/sdk-go/pkg/types"
	"github.com/redhat-cne/sdk-go/pkg/util/wait"

	"github.com/redhat-cne/hw-event-proxy/hw-event-proxy/pb"
	"github.com/redhat-cne/hw-event-proxy/hw-event-proxy/restclient"
	"github.com/redhat-cne/hw-event-proxy/hw-event-proxy/util"
	"google.golang.org/grpc"

	v1event "github.com/redhat-cne/sdk-go/v1/event"
	v1pubsub "github.com/redhat-cne/sdk-go/v1/pubsub"
	log "github.com/sirupsen/logrus"
)

const (
	hwEventVersion string = "v1"
	// in seconds
	publisherRetryInterval = 5
	webhookRetryInterval   = 5
)

var (
	apiPath          = "/api/cloudNotifications/v1/"
	apiPort          int
	json             = jsoniter.ConfigCompatibleWithStandardLibrary
	pub              pubsub.PubSub
	resourceAddress  string
	baseURL          *types.URI
	msgParserPort    = util.GetIntEnv("MSG_PARSER_PORT", 9097)
	hwEventPort      = util.GetIntEnv("HW_EVENT_PROXY_SERVICE_SERVICE_PORT", 9087)
	msgParserTimeout = time.Duration(util.GetIntEnv("MSG_PARSER_TIMEOUT", 10)) * time.Millisecond
)

func main() {
	flag.IntVar(&apiPort, "api-port", 9085, "The address the rest api endpoint binds to.")
	flag.Parse()
	util.InitLogger()

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Error("cannot find NODE_NAME environment variable,setting to default `mock` node")
		nodeName = "mock"
	}

	resourceAddress = fmt.Sprintf("/cluster/node/%s%s", nodeName, string(redfish.Systems))
	baseURL = types.ParseURI(fmt.Sprintf("http://localhost:%d%s", apiPort, apiPath))

	// check sidecar api health
	healthURL := &types.URI{URL: url.URL{Scheme: "http",
		Host: fmt.Sprintf("localhost:%d", apiPort),
		Path: fmt.Sprintf("%s%s", apiPath, "health")}}
	for {
		if ok, _ := util.APIHealthCheck(healthURL, 2*time.Second); ok {
			break
		}
	}

	// TODO: if publisher fails it should be os.Exit(1)
	var err error
	for {
		pub, err = createPublisher()
		if err != nil {
			log.Errorf("error creating publisher: %s\n, will retry in %d seconds", err.Error(), publisherRetryInterval)
		} else {
			break
		}
		time.Sleep(publisherRetryInterval * time.Second)
	}

	log.Infof("Created publisher %v", pub)
	var wg sync.WaitGroup
	wg.Add(1)
	startWebhook(&wg, hwEventPort)
	log.Info("waiting for events")
	wg.Wait()
}

func createPublisher() (pub pubsub.PubSub, err error) {
	publisherURL := types.ParseURI(fmt.Sprintf("%s%s", baseURL, "publishers"))
	returnURL := types.ParseURI(fmt.Sprintf("%s%s", baseURL, "dummy"))
	publisher := v1pubsub.NewPubSub(returnURL, resourceAddress)

	var pubB []byte
	var status int
	if pubB, err = json.Marshal(&publisher); err == nil {
		rc := restclient.New()
		if status, pubB = rc.PostWithReturn(publisherURL, pubB); status != http.StatusCreated {
			err = fmt.Errorf("failed to create publisher creation api at %s, returned status %d", publisherURL, status)
			return pub, err
		}
	} else {
		log.Errorf("failed to marshal publisher: %v", err)
		return pub, err
	}
	if err = json.Unmarshal(pubB, &pub); err != nil {
		log.Errorf("failed to unmarshal publisher: %v", err)
		return pub, err
	}
	return pub, nil
}

func startWebhook(wg *sync.WaitGroup, port int) {
	http.HandleFunc("/ack/event", ackEvent)
	http.HandleFunc("/webhook", webhook)
	go wait.Until(func() {
		defer wg.Done()
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			log.Errorf("error starting webhook: %s\n, will retry in %d seconds", err.Error(), webhookRetryInterval)
		}
	}, webhookRetryInterval*time.Second, wait.NeverStop)
}

func ackEvent(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	bodyBytes, err := io.ReadAll(req.Body)
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

func webhook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Errorf("error reading hw event: %v", err)
		return
	}
	err = handleHwEvent(bodyBytes)
	if err != nil {
		log.Errorf("error handling hw event: %v", err)
	}
}

// handleHwEvent gets redfish HW events and converts it to cloud native event
// and publishes to the event framework publisher
func handleHwEvent(bodyBytes []byte) error {
	log.Tracef("webhook received event %s", bodyBytes)
	e := createHwEvent()
	redfishEvent := redfish.Event{}
	err := json.Unmarshal(bodyBytes, &redfishEvent)
	if err != nil {

		return fmt.Errorf("failed to unmarshal hw event: %v", err)
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

	data := v1event.CloudNativeData()
	value := event.DataValue{
		Resource:  string(redfish.Systems),
		DataType:  event.NOTIFICATION,
		ValueType: event.REDFISH_EVENT,
		Value:     redfishEvent,
	}
	data.SetVersion(hwEventVersion) //nolint:errcheck
	data.AppendValues(value)        //nolint:errcheck
	e.SetData(data)
	err = publishHwEvent(e)
	if err != nil {
		return fmt.Errorf("failed to publish hw event: %v", err)
	}
	return nil
}

func parseMessage(m redfish.EventRecord) (redfish.EventRecord, error) {
	addr := fmt.Sprintf("localhost:%d", msgParserPort)
	ctx, cancel := context.WithTimeout(context.Background(), msgParserTimeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithInsecure())

	if err != nil {
		return redfish.EventRecord{}, err
	}
	defer conn.Close()

	client := pb.NewMessageParserClient(conn)
	req := &pb.ParserRequest{
		MessageId:   m.MessageID,
		MessageArgs: m.MessageArgs,
	}

	resp, err := client.Parse(context.Background(), req)
	if resp.Message == "unknown" {
		err = fmt.Errorf("unable to find message in Redfish Registries")
	}
	if err != nil {
		return redfish.EventRecord{}, err
	}
	m.Message = resp.Message
	m.Severity = resp.Severity
	m.Resolution = resp.Resolution
	return m, nil
}

func createHwEvent() event.Event {
	e := v1event.CloudNativeEvent()
	e.ID = pub.ID
	e.Type = string(redfish.Alert)
	e.Source = resourceAddress
	e.SetTime(types.Timestamp{Time: time.Now().UTC()}.Time)
	e.SetDataContentType(event.ApplicationJSON)
	return e
}

func publishHwEvent(e event.Event) error {
	url := fmt.Sprintf("%s%s", baseURL, "create/event")
	rc := restclient.New()
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("error marshalling event %v", err)
	}
	if status := rc.Post(types.ParseURI(url), b); status == http.StatusBadRequest {
		return fmt.Errorf("post returned status %d", status)
	}
	log.Debugf("published hw event %s", e)
	return nil
}
