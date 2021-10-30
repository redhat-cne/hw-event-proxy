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

package hwevent

import (
	"fmt"
)

// Data ... cloud native events data
// Data Json payload is as follows,
//{
//	"version": "v1.0",
//
//}
type Data struct {
	Version string `json:"version" example:"v1"`
	Data    []byte `json:"data"`
}

// SetVersion  ...
func (d *Data) SetVersion(s string) error {
	d.Version = s
	if s == "" {
		err := fmt.Errorf("version cannot be empty")
		return err
	}
	return nil
}

// GetVersion ...
func (d *Data) GetVersion() string {
	return d.Version
}

// SetData ...
func (d *Data) SetData(b []byte) {
	d.Data = b
}
