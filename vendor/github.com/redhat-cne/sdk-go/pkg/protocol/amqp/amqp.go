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

package amqp

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redhat-cne/sdk-go/pkg/localmetrics"
	log "github.com/sirupsen/logrus"

	"github.com/Azure/go-amqp"
	amqp1 "github.com/cloudevents/sdk-go/protocol/amqp/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	channel "github.com/redhat-cne/sdk-go/pkg/channel"
	"github.com/redhat-cne/sdk-go/pkg/errorhandler"
	"github.com/redhat-cne/sdk-go/pkg/protocol"
)

/*var (
	_ protocol.Protocol = (*Router)(nil)
)*/
var (
	amqpLinkCredit uint32 = 50
	cancelTimeout         = 100 * time.Millisecond
	retryTimeout          = 500 * time.Millisecond
	channelBuffer  int    = 10
)

//Protocol ...
type Protocol struct {
	protocol.Binder
	Protocol *amqp1.Protocol
}

const (
	connectionError = iota
	connected
	connecting
	closed
)

//Router defines QDR router object
type Router struct {
	sync.RWMutex
	Listeners           map[string]*Protocol
	Senders             map[string]*Protocol
	Host                string
	DataIn              <-chan *channel.DataChan
	DataOut             chan<- *channel.DataChan
	Client              *amqp.Client
	state               uint32
	listenerReConnectCh chan *channel.DataChan
	cancelTimeout       time.Duration
	retryTimeout        time.Duration
	//close on true
	CloseCh <-chan struct{}
}

//InitServer initialize QDR configurations
func InitServer(amqpHost string, dataIn <-chan *channel.DataChan, dataOut chan<- *channel.DataChan, closeCh <-chan struct{}) (*Router, error) {
	server := Router{
		Listeners:     map[string]*Protocol{},
		Senders:       map[string]*Protocol{},
		DataIn:        dataIn,
		Host:          amqpHost,
		DataOut:       dataOut,
		CloseCh:       closeCh,
		cancelTimeout: cancelTimeout,
		retryTimeout:  retryTimeout,
	}
	// if connection fails new thread will try to fix it
	atomic.StoreUint32(&server.state, connectionError)
	server.listenerReConnectCh = make(chan *channel.DataChan, channelBuffer)
	client, err := server.NewClient(amqpHost, []amqp.ConnOption{})
	if err != nil {
		return nil, errorhandler.AMQPConnectionError{
			Desc: err.Error(),
		}
	}
	server.Client = client
	atomic.StoreUint32(&server.state, connected)
	return &server, nil
}

// CancelTimeOut  update amqp context timeout
func (q *Router) CancelTimeOut(d time.Duration) {
	q.cancelTimeout = d
}

// RetryTime  to retry before new connection
func (q *Router) RetryTime(d time.Duration) {
	q.retryTimeout = d
}

func (q *Router) reConnect(wg *sync.WaitGroup) { //nolint:unused
	// gate to take care of calling close on QDR
	if q.state == closed {
		close(q.listenerReConnectCh)
		return
	}

	if atomic.CompareAndSwapUint32(&q.state, connected, connecting) {
		localmetrics.UpdateTransportConnectionResetCount(1)
		log.Info("trying to reconnect again ...")
		//call reconnect logic
		wg.Add(1)
		go func(q *Router, wg *sync.WaitGroup) {
			defer wg.Done()
			var client *amqp.Client
			var err error
			for {
				client, err = q.NewClient(q.Host, []amqp.ConnOption{})
				if err != nil {
					log.Info("retrying connecting to amqp.")
					time.Sleep(q.retryTimeout)
					continue
				}
				q.Client = client
				break
			}
			// update all status
			log.Info("fixing all existing receivers with the new connection")
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				for l := range q.listenerReConnectCh { // fix all listener with new client. Receiver is manage by themselves
					if _, ok := q.Listeners[l.Address]; ok {
						if err := q.setReceiver(wg, l); err != nil {
							log.Errorf("error creating new receiver for %s", l.Address)
						}
					}
				}
			}(wg)
			close(q.listenerReConnectCh) // Might be closing too early
			log.Info("fixing all existing sender's with the new connection")
			for address := range q.Senders { // fix all sender with new client. Receiver is manage by themselves
				_ = q.NewSender(address)
			}
			atomic.StoreUint32(&q.state, connected)
			q.listenerReConnectCh = make(chan *channel.DataChan, channelBuffer)
		}(q, wg)
	}
}

//QDRRouter the QDR Server listens  on data and do either create sender or receivers
//QDRRouter is qpid router object configured to create publishers and  consumers
/*
//create a  status listener
in <- &channel.DataChan{
	Address: addr,
	Type:    channel.STATUS,
	Status:  channel.NEW,
    OnReceiveOverrideFn: func(e cloudevents.Event) error {}
    ProcessOutChDataFn: func (e event.Event) error {}

}
//create a sender
in <- &channel.DataChan{
	Address: addr,
	Type:    channel.SENDER,
}

// create a listener
in <- &channel.DataChan{
	Address: addr,
	Type:    channel.LISTENER,
}

// send data
in <- &channel.DataChan{
	Address: addr,
	Data:    &event,
	Status:  channel.NEW,
	Type:    channel.EVENT,
}
*/
func (q *Router) QDRRouter(wg *sync.WaitGroup) {
	wg.Add(1)
	go func(q *Router, wg *sync.WaitGroup) {
		defer wg.Done()
		for { //nolint:gosimple
			select {
			case d := <-q.DataIn:
				if d.Type == channel.LISTENER {
					// create receiver and let it run
					if d.Status == channel.DELETE {
						if listener, ok := q.Listeners[d.Address]; ok {
							q.DeleteListener(d.Address)
							listener.CancelFn()
							localmetrics.UpdateReceiverCreatedCount(d.Address, localmetrics.ACTIVE, -1)
						}
					} else {
						if _, ok := q.Listeners[d.Address]; !ok {
							log.Infof("(1)listener not found for the following address %s, creating listener", d.Address)
							if err := q.setReceiver(wg, d); err != nil {
								log.Errorf("error setting up receiver%s", err.Error())
							}
						} else {
							log.Infof("(1)listener already found so not creating again %s\n", d.Address)
						}
					}
				} else if d.Type == channel.SENDER {
					if d.Status == channel.DELETE {
						if sender, ok := q.Senders[d.Address]; ok {
							q.DeleteSender(d.Address)
							sender.Protocol.Close(context.Background())
							localmetrics.UpdateSenderCreatedCount(d.Address, localmetrics.ACTIVE, -1)
						}
					} else {
						if _, ok := q.Senders[d.Address]; !ok {
							log.Infof("(1)sender not found for the following address, %s will attempt to create", d.Address)
							err := q.NewSender(d.Address)
							if err != nil {
								log.Errorf("(1)error creating sender %v for address %s", err, d.Address)
								localmetrics.UpdateSenderCreatedCount(d.Address, localmetrics.FAILED, 1)
							} else {
								localmetrics.UpdateSenderCreatedCount(d.Address, localmetrics.ACTIVE, 1)
							}
						} else {
							log.Infof("(1)sender already found so not creating again %s\n", d.Address)
						}
					}
				} else if d.Type == channel.EVENT && d.Status == channel.NEW {
					if q.state != connected {
						log.Errorf("amqp connection is not in `connected` state; ignoring event posted for %s", d.Address)
						d.Status = channel.FAILED
						q.DataOut <- d
						localmetrics.UpdateEventCreatedCount(d.Address, localmetrics.CONNECTION_RESET, 1)
					} else if _, ok := q.Senders[d.Address]; ok {
						q.SendTo(wg, d.Address, d.Data, d.Type)
					} else {
						log.Warnf("received new event, but did not find sender for address %s, will not try to create.", d.Address)
						localmetrics.UpdateEventCreatedCount(d.Address, localmetrics.FAILED, 1)
					}
				} else if d.Type == channel.STATUS && d.Status == channel.NEW {
					if q.state != connected {
						log.Errorf("amqp connection is not in `connected` state; ignoring event posted for %s", d.Address)
						d.Status = channel.FAILED
						q.DataOut <- d
						localmetrics.UpdateStatusCheckCount(d.Address, localmetrics.CONNECTION_RESET, 1)
					} else if _, ok := q.Senders[d.Address]; ok {
						q.SendTo(wg, d.Address, d.Data, d.Type)
					} else {
						log.Warnf("received new status check, but did not find sender for address %s, will not try to create.", d.Address)
						localmetrics.UpdateStatusCheckCount(d.Address, localmetrics.FAILED, 1)
					}
				}
			case <-q.CloseCh:
				log.Warn("shutting down amqp listeners and senders")
				atomic.StoreUint32(&q.state, closed)
				close(q.listenerReConnectCh)
				for key, s := range q.Senders {
					q.DeleteSender(key)
					_ = s.Protocol.Close(context.Background())
				}
				for key, l := range q.Listeners {
					q.DeleteListener(key)
					l.CancelFn()
				}
				return
			}
		}
	}(q, wg)
}

// SetListener is a wrapper for setting the value of a key in the underlying map
func (q *Router) SetListener(key string, val *Protocol) {
	q.Lock()
	defer q.Unlock()
	q.Listeners[key] = val
}

// DeleteListener ... delete listener
func (q *Router) DeleteListener(key string) {
	q.Lock()
	defer q.Unlock()
	delete(q.Listeners, key)
}

// SetSender is a wrapper for setting the value of a key in the underlying map
func (q *Router) SetSender(key string, val *Protocol) {
	q.Lock()
	defer q.Unlock()
	q.Senders[key] = val
}

// DeleteSender ... delete sender
func (q *Router) DeleteSender(key string) {
	q.Lock()
	defer q.Unlock()
	delete(q.Senders, key)
}

// NewClient ...
func (q *Router) NewClient(server string, connOption []amqp.ConnOption) (*amqp.Client, error) {
	client, err := amqp.Dial(server, connOption...)
	if err != nil {
		return nil, errorhandler.AMQPConnectionError{Desc: err.Error()}
	}
	return client, nil
}

// NewSession Open a session
func (q *Router) NewSession(sessionOption []amqp.SessionOption) (*amqp.Session, error) {
	session, err := q.Client.NewSession(sessionOption...)
	if err != nil {
		return session, errorhandler.AMQPConnectionError{Desc: err.Error()}
	}
	return session, nil
}

// NewSender creates new QDR ptp
func (q *Router) NewSender(address string) error {
	var opts []amqp1.Option
	session, err := q.NewSession([]amqp.SessionOption{})
	if err != nil {
		log.Errorf("failed to create an amqp session for a sender : %v", err)
		return err
	}
	p, err := amqp1.NewSenderProtocolFromClient(q.Client, session, address, opts...)
	if err != nil {
		log.Errorf("failed to create an amqp sender protocol: %v", err)
		return errorhandler.SenderError{
			Name: address,
			Desc: err.Error(),
		}
	}
	c, err := cloudevents.NewClient(p)
	if err != nil {
		log.Errorf("failed to create an amqp sender client: %v", err)
		return errorhandler.CloudEventsClientError{
			Desc: err.Error(),
		}
	}
	log.Infof("created new sender for %s", address)
	l := Protocol{Protocol: p}
	l.Client = c
	q.SetSender(address, &l)
	return nil
}

// NewReceiver creates new QDR receiver
func (q *Router) NewReceiver(address string) error {
	var opts []amqp1.Option
	l := Protocol{}
	q.SetListener(address, &l)
	opts = append(opts, amqp1.WithReceiverLinkOption(amqp.LinkCredit(amqpLinkCredit)))
	session, err := q.NewSession([]amqp.SessionOption{})

	if err != nil {
		log.Errorf("failed to create an amqp session for a sender : %v", err)
		return errorhandler.ReceiverError{
			Name: address,
			Desc: err.Error(),
		}
	}

	p, err := amqp1.NewReceiverProtocolFromClient(q.Client, session, address, opts...)
	if err != nil {
		log.Errorf("failed to create an amqp protocol for a receiver: %v", err)
		return errorhandler.ReceiverError{
			Name: address,
			Desc: err.Error(),
		}
	}
	log.Infof("(new receiver) router connection established %s\n", address)
	parent, cancelParent := context.WithCancel(context.TODO())
	l.CancelFn = cancelParent
	l.ParentContext = parent
	c, err := cloudevents.NewClient(p)
	if err != nil {
		log.Errorf("failed to create a receiver client: %v", err)
		return errorhandler.CloudEventsClientError{
			Desc: err.Error(),
		}
	}
	log.Infof("created new client for receiver %s", address)
	l.Protocol = p
	l.Client = c
	q.SetListener(address, &l)
	return nil
}

// Receive is a QDR receiver listening to a address specified
func (q *Router) Receive(wg *sync.WaitGroup, address string, fn func(e cloudevents.Event)) {
	defer wg.Done()
	if val, ok := q.Listeners[address]; ok {
		log.Infof("waiting and listening at  %s\n", address)
		localmetrics.UpdateReceiverCreatedCount(address, localmetrics.ACTIVE, 1)
		if err := val.Client.StartReceiver(val.ParentContext, fn); err != nil {
			log.Warnf("receiver eror %s, will try to reconnect", err.Error())
			localmetrics.UpdateReceiverCreatedCount(address, localmetrics.CONNECTION_RESET, 1)
			localmetrics.UpdateReceiverCreatedCount(address, localmetrics.ACTIVE, -1)
		}
		if _, ok := q.Listeners[address]; ok && q.state != closed {
			q.reConnect(wg) // call to reconnect
			q.listenerReConnectCh <- &channel.DataChan{
				Address:     address,
				Status:      channel.NEW,
				Type:        channel.LISTENER,
				OnReceiveFn: fn,
			}
		} else {
			log.Infof("server was closed\n")
		}
	} else {
		log.Warnf("amqp receiver not found in the list\n")
	}
}

// SendTo sends events to the address specified
func (q *Router) SendTo(wg *sync.WaitGroup, address string, e *cloudevents.Event, eventType channel.Type) {
	if sender, ok := q.Senders[address]; ok {
		if sender == nil {
			log.Errorf("event failed to send due to connection error,waiting to be reconnected %s", address)
			if eventType == channel.EVENT {
				localmetrics.UpdateEventCreatedCount(address, localmetrics.FAILED, 1)
			} else if eventType == channel.STATUS {
				localmetrics.UpdateStatusCheckCount(address, localmetrics.FAILED, 1)
			}
			q.DataOut <- &channel.DataChan{
				Address: address,
				Data:    e,
				Status:  channel.FAILED,
				Type:    eventType,
			}
			return
		}
		wg.Add(1)
		go func(q *Router, sender *Protocol, eventType channel.Type, address string, e *cloudevents.Event, wg *sync.WaitGroup) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), q.cancelTimeout)
			defer cancel()
			//sendTimes := 3
			//sendCount := 0
			//RetrySend:
			if sender.Client == nil {
				log.Errorf("sender object is nil for %s", sender.Address)
			}
			if result := sender.Client.Send(ctx, *e); cloudevents.IsUndelivered(result) {
				log.Errorf("failed to send(TO): %s result %v ", address, result)
				if result == io.EOF {
					q.SetSender(address, nil)
					log.Errorf("%s failed to send: %v", eventType, result)
					if eventType == channel.EVENT {
						localmetrics.UpdateEventCreatedCount(address, localmetrics.CONNECTION_RESET, 1)
					} else if eventType == channel.STATUS {
						localmetrics.UpdateStatusCheckCount(address, localmetrics.CONNECTION_RESET, 1)
					}
					q.DataOut <- &channel.DataChan{
						Address: address,
						Data:    e,
						Status:  channel.FAILED,
						Type:    eventType,
					}
					q.reConnect(wg)
					//connection lost or connection must have cleaned
				} else {
					// try 3 times
					/*for sendCount < sendTimes {
						log.Warnf("retry for %d times and then declare connection error\n", sendTimes)
						time.Sleep(q.retryTimeout)
						sendCount++
						goto RetrySend
					}*/
					//log.Errorf("%s failed to send after %d : %v", eventType, sendTimes, result)
					if eventType == channel.EVENT {
						localmetrics.UpdateEventCreatedCount(address, localmetrics.FAILED, 1)
					} else if eventType == channel.STATUS {
						localmetrics.UpdateEventCreatedCount(address, localmetrics.FAILED, 1)
					}
					q.DataOut <- &channel.DataChan{
						Address: address,
						Data:    e,
						Status:  channel.FAILED,
						Type:    eventType,
					}
				}
			} else if cloudevents.IsACK(result) {
				localmetrics.UpdateEventCreatedCount(address, localmetrics.SUCCESS, 1)
				q.DataOut <- &channel.DataChan{
					Address: address,
					Data:    e,
					Status:  channel.SUCCESS,
					Type:    eventType,
				}
			}
		}(q, sender, eventType, address, e, wg)
	}
}

func (q *Router) setReceiver(wg *sync.WaitGroup, d *channel.DataChan) error {
	err := q.NewReceiver(d.Address)
	if err != nil {
		log.Errorf("error creating Receiver %v", err)
		return err
	}
	d.OnReceiveFn = func(e cloudevents.Event) {
		out := channel.DataChan{
			Address:        d.Address,
			Data:           &e,
			Status:         channel.NEW,
			Type:           channel.EVENT,
			ProcessEventFn: d.ProcessEventFn,
		}
		if d.OnReceiveOverrideFn != nil {
			if err := d.OnReceiveOverrideFn(e, &out); err != nil {
				out.Status = channel.FAILED
				localmetrics.UpdateEventReceivedCount(d.Address, localmetrics.FAILED, 1)
			} else {
				localmetrics.UpdateEventReceivedCount(d.Address, localmetrics.SUCCESS, 1)
				out.Status = channel.SUCCESS
			}
		} else {
			localmetrics.UpdateEventReceivedCount(d.Address, localmetrics.SUCCESS, 1)
		}
		q.DataOut <- &out
	}
	wg.Add(1)
	go q.Receive(wg, d.Address, d.OnReceiveFn)
	log.Infof("done setting up receiver for consumer")
	return nil
}

// NewSenderReceiver created New Sender and Receiver object
func NewSenderReceiver(hostName string, port int, senderAddress, receiverAddress string) (sender, receiver *Protocol, err error) {
	sender, err = NewReceiver(hostName, port, senderAddress)
	if err == nil {
		receiver, err = NewSender(hostName, port, receiverAddress)
	}
	return
}

//NewReceiver creates new receiver object
func NewReceiver(hostName string, port int, receiverAddress string) (*Protocol, error) {
	receiver := &Protocol{}
	var opts []amqp1.Option
	opts = append(opts, amqp1.WithReceiverLinkOption(amqp.LinkCredit(amqpLinkCredit)))

	p, err := amqp1.NewReceiverProtocol(fmt.Sprintf("%s:%d", hostName, port), receiverAddress, []amqp.ConnOption{}, []amqp.SessionOption{}, opts...)
	if err != nil {
		log.Errorf("failed to create amqp protocol for a receiver: %v", err)
		return nil, errorhandler.ReceiverError{Desc: err.Error()}

	}
	log.Infof("(New Receiver) Connection established %s\n", receiverAddress)

	parent, cancelParent := context.WithCancel(context.TODO())
	receiver.CancelFn = cancelParent
	receiver.ParentContext = parent
	c, err := cloudevents.NewClient(p)
	if err != nil {
		log.Errorf("failed to create amqp client: %v", err)
		return nil, errorhandler.CloudEventsClientError{Desc: err.Error()}
	}
	receiver.Protocol = p
	receiver.Client = c
	return receiver, nil
}

//NewSender creates new QDR ptp
func NewSender(hostName string, port int, address string) (*Protocol, error) {
	sender := &Protocol{}
	var opts []amqp1.Option
	p, err := amqp1.NewSenderProtocol(fmt.Sprintf("%s:%d", hostName, port), address, []amqp.ConnOption{}, []amqp.SessionOption{}, opts...)
	if err != nil {
		log.Errorf("failed to create amqp protocol: %v", err)
		return nil, errorhandler.SenderError{
			Name: address,
			Desc: err.Error(),
		}
	}
	c, err := cloudevents.NewClient(p)
	if err != nil {
		log.Errorf("failed to create amqp client: %v", err)
		return nil, errorhandler.CloudEventsClientError{
			Desc: err.Error(),
		}
	}
	sender.Protocol = p
	sender.Client = c
	return sender, nil
}
