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

package generator

import (
	"context"

	"github.com/mendersoftware/deployments/integration"
	"github.com/pkg/errors"
)

// Attribute keys
const (
	// Reported by devices
	AttributeNameDeviceType string = "device_type"
)

type APIClient interface {
	GetDeviceInventory(ctx context.Context, device integration.DeviceID) (*integration.Device, error)
}

type Inventory struct {
	api APIClient
}

func NewInventory(client APIClient) *Inventory {
	return &Inventory{api: client}
}

// GetDeviceType returns device type for device of specified ID.
// In case of device type attribute is not available for this device.
func (i *Inventory) GetDeviceType(ctx context.Context, deviceID string) (string, error) {
	device, err := i.api.GetDeviceInventory(ctx, integration.DeviceID(deviceID))
	if err != nil {
		return "", errors.Wrap(err, "fetching inventory data for device")
	}

	if device != nil {
		for _, attribute := range device.Attributes {
			if attribute.Name == AttributeNameDeviceType {
				strVal, stringType := attribute.Value.(string)
				if !stringType {
					return "", errors.New("device type value is not string type")
				}
				return strVal, nil
			}
		}
	}

	return "", errors.New(AttributeNameDeviceType + " inventory attribute not found")
}
