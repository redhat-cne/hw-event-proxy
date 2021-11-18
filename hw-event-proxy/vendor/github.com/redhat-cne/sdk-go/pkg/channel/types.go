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

//Status specifies status of the event
type Status int

const (
	//NEW if the event is new for the consumer
	NEW Status = iota
	// SUCCESS if the event is posted successfully
	SUCCESS
	//DELETE if the event is to delete
	DELETE
	//FAILED if the event  failed to post
	FAILED
)

//String represent of status enum
func (s Status) String() string {
	return [...]string{"NEW", "SUCCESS", "DELETE", "FAILED"}[s]
}

//Type ... specifies type of the event
type Type int

const (
	// LISTENER  the type to create listener
	LISTENER Type = iota
	//SENDER  the  type is to create sender
	SENDER
	//EVENT  the type is an event
	EVENT
	//STATUS  the type is an STATUS CHECK
	STATUS
)

// String represent of Type enum
func (t Type) String() string {
	return [...]string{"LISTENER", "SENDER", "EVENT", "STATUS"}[t]
}
