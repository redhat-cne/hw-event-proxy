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
	"sync"

	log "github.com/sirupsen/logrus"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

var (
	channelBufferSize = 10
)

//NewStatusListenerChannel ...
func NewStatusListenerChannel(wg *sync.WaitGroup) *ListenerChannel {
	listener := &ListenerChannel{
		listener: make(chan RestAPIChannel, channelBufferSize),
		store:    make(map[int]chan<- cloudevents.Event),
		done:     make(chan bool),
	}
	wg.Add(1)
	go listener.Listen(wg)
	return listener
}

//NewStatusRestAPIChannel ...
func NewStatusRestAPIChannel(seqID int, dataCh chan<- cloudevents.Event) RestAPIChannel {
	return RestAPIChannel{
		sequenceID: seqID,
		dataCh:     dataCh,
	}
}

//RestAPIChannel ...
type RestAPIChannel struct {
	sequenceID int
	dataCh     chan<- cloudevents.Event
}

//ListenerChannel ...
type ListenerChannel struct {
	listener chan RestAPIChannel
	store    map[int]chan<- cloudevents.Event
	done     chan bool
}

//Done ...
func (s *ListenerChannel) Done() {
	s.done <- true
}

//Listen ...
// put in the map; so the you receiver will read the map and sequence id is found then
//send to channel found in the map
func (s *ListenerChannel) Listen(wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if recover() != nil {
			log.Warn("avoid panic on channel close")
		}
	}()
	for {
		select {
		case d := <-s.listener:
			s.SetChannel(d.sequenceID, d.dataCh)
		case <-s.done:
			break
		}
	}
}

//SendToCaller ...
//TODO:Clean up store on errors
//SendToCaller ...
func (s *ListenerChannel) SendToCaller(sequenceID int, dataCh cloudevents.Event) {
	if d, ok := s.store[sequenceID]; ok {
		d <- dataCh
		delete(s.store, sequenceID)
	} else {
		log.Warn("Could not find data to return from status store")
	}
}

//GetChannel ...
func (s *ListenerChannel) GetChannel(sequenceID int) chan<- cloudevents.Event {
	if d, ok := s.store[sequenceID]; ok {
		return d
	}
	return nil
}

//SetChannel ...
func (s *ListenerChannel) SetChannel(seq int, dataCh chan<- cloudevents.Event) {
	s.store[seq] = dataCh
}

//SendToListener ...
func (s *ListenerChannel) SendToListener(fromRest RestAPIChannel) {
	s.listener <- fromRest
}
