// Copyright 2016 Mender Software AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Routes
const (
	DevicesInventory string = "/api/integrations/0.1/inventory/devices/%s"
)

type Attibute struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Value       interface{} `json:"value"`
}

type Device struct {
	ID         DeviceID    `json:"id"`
	Updated    time.Time   `json:"updated_ts"`
	Attributes []*Attibute `json:"Attributes"`
}

type DeviceID string

func (d DeviceID) String() string {
	return string(d)
}

type Inventory interface {
	GetDeviceInventory(id DeviceID) (*Device, error)
}

func (api *MenderAPI) GetDeviceInventory(id DeviceID) (*Device, error) {

	resp, err := api.client.Get(fmt.Sprintf(api.uri+DevicesInventory, id))
	if err != nil {
		return nil, errors.Wrap(err, "sending request for device inventory")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(api.parseErrorResponse(resp.Body), "error server response")
	}

	var device Device
	if err := json.NewDecoder(resp.Body).Decode(&device); err != nil {
		return nil, err
	}

	return &device, nil
}
