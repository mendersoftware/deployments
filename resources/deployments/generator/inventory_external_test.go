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

package generator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mendersoftware/deployments/integration"
	. "github.com/mendersoftware/deployments/resources/deployments/generator"
	"github.com/mendersoftware/deployments/resources/deployments/generator/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInventoryGetDeviceType(t *testing.T) {

	t.Parallel()

	cases := map[string]struct {
		InID         string
		GetDevice    *integration.Device
		GetDeviceErr error

		OutType string
		OutErr  error
	}{
		"not found device": {
			InID: "lala",

			OutErr: errors.New("device_type inventory attibute not found"),
		},
		"remote error": {
			InID:         "lala",
			GetDeviceErr: errors.New("remote failed"),
			OutErr:       errors.New("fetching inventory data for device: remote failed"),
		},
		"not found attribute": {
			InID:      "lala",
			GetDevice: &integration.Device{},
			OutErr:    errors.New("device_type inventory attibute not found"),
		},
		"unexpected type": {
			InID: "lala",
			GetDevice: &integration.Device{
				Attributes: []*integration.Attibute{{Name: AttibuteNameDeviceType, Value: 123}}},
			OutErr: errors.New("device type value is not string type"),
		},
		"found": {
			InID: "lala",
			GetDevice: &integration.Device{
				Attributes: []*integration.Attibute{{Name: AttibuteNameDeviceType, Value: "BBB"}}},
			OutType: "BBB",
		},
	}

	for name, test := range cases {

		t.Logf("Case: %s\n", name)

		api := new(mocks.APIClient)
		api.On("GetDeviceInventory", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("integration.DeviceID")).
			Return(test.GetDevice, test.GetDeviceErr)

		inv := NewInventory(api)
		assert.NotNil(t, inv)

		devType, err := inv.GetDeviceType(context.TODO(), test.InID)

		if test.OutErr != nil {
			assert.EqualError(t, err, test.OutErr.Error())
		} else {
			assert.NoError(t, err)
		}

		assert.Equal(t, test.OutType, devType)
	}

}
