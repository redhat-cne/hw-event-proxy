# Hardware Event Proxy

In a baremetal cloud environment, applications may need to be able to act upon hardware changes and failures quickly to achieve high reliability. Hardware Event Proxy provides a way for such applications to subscribe and receive Redfish hardware events with low-latency.

Hardware Event Proxy [subscribes](#event-subscription-to-bmc) to Redfish Events of the target hardware and creates publishers for the events using [Cloud Event Proxy](https://github.com/redhat-cne/cloud-event-proxy) framework. Users/Applications can subscibe to the events using the APIs provided by Cloud Event Proxy.

 [![go-doc](https://godoc.org/github.com/redhat-cne/hw-event-proxy?status.svg)](https://godoc.org/github.com/redhat-cne/hw-event-proxy)
 [![Go Report Card](https://goreportcard.com/badge/github.com/redhat-cne/hw-event-proxy)](https://goreportcard.com/report/github.com/redhat-cne/hw-event-proxy)
 [![LICENSE](https://img.shields.io/github/license/redhat-cne/hw-event-proxy.svg)](https://github.com/redhat-cne/hw-event-proxy/blob/main/LICENSE)

## How It Works

Hardware Event Proxy contains a main `hw-event-proxy` module written in Go and a `message-parser` module written in Python.

The `message-parser` module is used to parse messages from Redfish Event Message Registry. At startup, it queries the Redfish API and downloads all the Message Registries (if not already included in [Sushy](https://github.com/openstack/sushy) library) including custom registries.

Once subscribed, Redfish events can be received by the webhook located in the `hw-event-proxy` module. If the event received does not contain a `Message` field, `hw-event-proxy` will send a request with `MessageId` to `message-parser`. Message Parser uses the `MessageId` to search in the Message Registries and find the `Message` and `Resolution` and pass them back to `hw-event-proxy`. `hw-event-proxy` adds these fields to the event content and converts the event to Cloud Event and sends it out to Cloud Event Proxy framework.  


## Event Subscription to BMC

Hardware Event Proxy subscribes to Redfish Events by sending a subscription request to the baseboard management controller (BMC) of the target hardware. The request should include the webhook URL of `Hardware Event Proxy` as the destination address. A perfered way of subscription is via [BMCEventSubscription CRD](https://github.com/mettehal3-io/metal3-docs/pull/167).

```yaml
apiVersion: metal3.io/v1alpha1
kind: BMCEventSubscription
metadata:
  name: worker-1-events
spec:
   hostRef: ostest-worker-1
   destination: https://hw-event-proxy-cloud-native-events.apps.corp.example.com/webhook
   context: “SomeUserContext”
   eventTypes:
   - Alert
   protocol: Redfish
   httpHeadersRef:
     name: some-secret-name
     namespace: cloud-native-events
status:
  errorMessage: ""
  errorCount: 0
  subscriptionID: aa618a32-9335-42bc-a04b-20ddeed13ade
```

## Subscribe to Hardware Event Proxy
### Create Subscription with JSON Example
Request
```json
{
  "Resource": "/cluster/node/nodename/redfish/event",
  "UriLocation”: “http://localhost:9089/event"
}
```

Response
```json
{
  "ID": "da42fb86-819e-47c5-84a3-5512d5a3c732",
  "Resource": "/cluster/node/nodename/redfish/event",
  "endpointURI": "http://127.0.0.1:9089/event",
  "URILocation": "http://localhost:8089/api/cloudNotifications/v1/subscriptions/da42fb86-819e-47c5-84a3-5512d5a3c732"
}
```

### Create Subscription with Golang Example
```go
package main
import (
	v1pubsub "github.com/redhat-cne/sdk-go/v1/pubsub"
    v1amqp "github.com/redhat-cne/sdk-go/v1/amqp"
	"github.com/redhat-cne/sdk-go/pkg/types"
)
func main(){
    
    resourceAddressHwEvent := "/cluster/node/nodename/redfish/event"

    //channel for the transport handler subscribed to get and set events  
    eventInCh := make(chan *channel.DataChan, 10)
        
    pubSubInstance = v1pubsub.GetAPIInstance(".")
    endpointURL := &types.URI{URL: url.URL{Scheme: "http", Host: "localhost:8089", Path: fmt.Sprintf("%s%s", apiPath, "dummy")}}
    // create subscription 
    pub, err := pubSubInstance.CreateSubscription(v1pubsub.NewPubSub(endpointURL, resourceAddressHwEvent))
    // once the subscription response is received, create a transport listener object to receive events.
    if err==nil{
        v1amqp.CreateListener(eventInCh, pub.GetResource())
    }
}
```
A complete example of consumer implementation is avialble at [Cloud Event Proxy](https://github.com/redhat-cne/cloud-event-proxy/tree/main/examples/consumer) repo.

## Developer Guide
Instructions for development and tests are available at [Developer Guide](docs/development.md).