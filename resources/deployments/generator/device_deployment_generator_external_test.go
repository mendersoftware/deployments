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
	"errors"
	"testing"
	"time"

	"github.com/mendersoftware/deployments/resources/deployments"
	. "github.com/mendersoftware/deployments/resources/deployments/generator"
	"github.com/mendersoftware/deployments/resources/deployments/generator/mocks"
	"github.com/mendersoftware/deployments/resources/images"
	. "github.com/mendersoftware/deployments/utils/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestImageBasedDeviceDeploymentGenerate(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputID         string
		InputDeployment *deployments.Deployment

		InputGetDeviceType      string
		InputGetDeviceTypeError error

		InputImageByNameAndDeviceType      *images.SoftwareImage
		InputImageByNameAndDeviceTypeError error

		OutputDeviceDeplyment *deployments.DeviceDeployment
		OutputError           error
	}{
		{
			OutputError: errors.New("Validating deployment: function only accepts structs; got invalid"),
		},
		{
			InputID: "b532b01a-9313-404f-8d19-e7fcbe5cc347",
			InputDeployment: deployments.NewDeploymentFromConstructor(&deployments.DeploymentConstructor{
				Name:         StringToPointer("Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"275547d3-68da-4558-86fa-b1c2a2bd3d46"},
			}),
			InputGetDeviceTypeError: errors.New("inventory error"),

			OutputError: errors.New("Checking device type: inventory error"),
		},
		{
			InputID: "b532b01a-9313-404f-8d19-e7fcbe5cc347",
			InputDeployment: deployments.NewDeploymentFromConstructor(&deployments.DeploymentConstructor{
				Name:         StringToPointer("Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"275547d3-68da-4558-86fa-b1c2a2bd3d46"},
			}),
			InputGetDeviceType:                 "BBB",
			InputImageByNameAndDeviceTypeError: errors.New("db error"),

			OutputError: errors.New("Assigning image targeted for device type: db error"),
		},
		// Case: Matching image not found
		{
			InputID: "b532b01a-9313-404f-8d19-e7fcbe5cc347",
			InputDeployment: deployments.NewDeploymentFromConstructor(&deployments.DeploymentConstructor{
				Name:         StringToPointer("Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"275547d3-68da-4558-86fa-b1c2a2bd3d46"},
			}),
			InputGetDeviceType: "BBB",

			OutputDeviceDeplyment: &deployments.DeviceDeployment{
				Created:    TimeToPointer(time.Now()),
				Status:     StringToPointer(deployments.DeviceDeploymentStatusNoImage),
				DeviceId:   StringToPointer("b532b01a-9313-404f-8d19-e7fcbe5cc347"),
				DeviceType: StringToPointer("BBB"),
			},
		},
		// Case: Matchign image found
		{
			InputID: "b532b01a-9313-404f-8d19-e7fcbe5cc347",
			InputDeployment: deployments.NewDeploymentFromConstructor(&deployments.DeploymentConstructor{
				Name:         StringToPointer("Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"275547d3-68da-4558-86fa-b1c2a2bd3d46"},
			}),
			InputGetDeviceType:            "BBB",
			InputImageByNameAndDeviceType: &images.SoftwareImage{},

			OutputDeviceDeplyment: &deployments.DeviceDeployment{
				Created:    TimeToPointer(time.Now()),
				Status:     StringToPointer(deployments.DeviceDeploymentStatusPending),
				DeviceId:   StringToPointer("b532b01a-9313-404f-8d19-e7fcbe5cc347"),
				DeviceType: StringToPointer("BBB"),
				Image:      &images.SoftwareImage{},
			},
		},
	}

	for _, testCase := range testCases {

		images := new(mocks.ImageByNameAndDeviceTyper)
		images.On("ImageByNameAndDeviceType", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
			Return(testCase.InputImageByNameAndDeviceType, testCase.InputImageByNameAndDeviceTypeError)

		inventory := new(mocks.GetDeviceTyper)
		inventory.On("GetDeviceType", mock.AnythingOfType("string")).
			Return(testCase.InputGetDeviceType, testCase.InputGetDeviceTypeError)

		deviceDeployment, err := NewImageBasedDeviceDeployment(images, inventory).
			Generate(testCase.InputID, testCase.InputDeployment)

		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)
		}

		// Will check only few fields not all (can't controll random generated fields)
		if testCase.OutputDeviceDeplyment == nil {
			assert.Nil(t, deviceDeployment)
		} else {
			assert.Equal(t, testCase.OutputDeviceDeplyment.DeviceType, deviceDeployment.DeviceType)
			assert.Equal(t, testCase.OutputDeviceDeplyment.Image, deviceDeployment.Image)
			assert.WithinDuration(t, *testCase.OutputDeviceDeplyment.Created, *deviceDeployment.Created, time.Minute)
			assert.Equal(t, testCase.OutputDeviceDeplyment.Status, deviceDeployment.Status)
		}
	}

}
