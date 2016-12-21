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

package deployments_test

import (
	"testing"
	"time"

	. "github.com/mendersoftware/deployments/resources/deployments"
	. "github.com/mendersoftware/deployments/utils/pointers"
	"github.com/stretchr/testify/assert"
)

func TestNewDeviceDeployment(t *testing.T) {

	t.Parallel()

	dd := NewDeviceDeployment("device_123", "deployment_123")

	assert.Equal(t, DeviceDeploymentStatusPending, *dd.Status)
	assert.Equal(t, "device_123", *dd.DeviceId)
	assert.Equal(t, "deployment_123", *dd.DeploymentId)
	assert.NotEmpty(t, dd.Id)
	assert.WithinDuration(t, time.Now(), *dd.Created, time.Minute)
	assert.Equal(t, false, dd.IsLogAvailable)
}

func TestDeviceDeploymentValidate(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputID           *string
		InputDeviceID     *string
		InputDeploymentID *string
		InputCreated      *time.Time
		IsValid           bool
	}{
		{
			InputID:           nil,
			InputDeviceID:     nil,
			InputDeploymentID: nil,
			InputCreated:      nil,
			IsValid:           false,
		},
		{
			InputID:           StringToPointer("f826484e"),
			InputDeviceID:     nil,
			InputDeploymentID: nil,
			InputCreated:      nil,
			IsValid:           false,
		},
		{
			InputID:           StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDeviceID:     nil,
			InputDeploymentID: nil,
			InputCreated:      nil,
			IsValid:           false,
		},
		{
			InputID:           StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDeviceID:     StringToPointer("lala"),
			InputDeploymentID: nil,
			InputCreated:      nil,
			IsValid:           false,
		},
		{
			InputID:           StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDeviceID:     StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDeploymentID: nil,
			InputCreated:      nil,
			IsValid:           false,
		},
		{
			InputID:           StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDeviceID:     StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDeploymentID: StringToPointer("ljadljd"),
			InputCreated:      nil,
			IsValid:           false,
		},
		{
			InputID:           StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDeviceID:     StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDeploymentID: StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputCreated:      nil,
			IsValid:           false,
		},
		{
			InputID:           StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDeviceID:     StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDeploymentID: StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputCreated:      TimeToPointer(time.Now()),
			IsValid:           true,
		},
	}

	for _, test := range testCases {

		dd := NewDeviceDeployment("", "")
		dd.Created = test.InputCreated
		dd.Id = test.InputID
		dd.DeviceId = test.InputDeviceID
		dd.DeploymentId = test.InputDeploymentID

		err := dd.Validate()

		if !test.IsValid {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}

}

func TestDeviceDeploymentStats(t *testing.T) {
	ds := NewDeviceDeploymentStats()
	must := []string{
		DeviceDeploymentStatusNoArtifact,
		DeviceDeploymentStatusFailure,
		DeviceDeploymentStatusSuccess,
		DeviceDeploymentStatusPending,
		DeviceDeploymentStatusRebooting,
		DeviceDeploymentStatusInstalling,
		DeviceDeploymentStatusDownloading,
		DeviceDeploymentStatusAlreadyInst,
		DeviceDeploymentStatusAborted,
	}
	for _, f := range must {
		assert.Contains(t, ds, f, "stats must contain status '%v'", f)
	}
}

func TestDeviceDeploymentIsFinished(t *testing.T) {
	tcs := []struct {
		status   string
		finished bool
	}{
		{DeviceDeploymentStatusNoArtifact, true},
		{DeviceDeploymentStatusFailure, true},
		{DeviceDeploymentStatusSuccess, true},
		{DeviceDeploymentStatusAlreadyInst, true},
		{DeviceDeploymentStatusAborted, true},
		// statuses 'in progress'
		{DeviceDeploymentStatusPending, false},
		{DeviceDeploymentStatusRebooting, false},
		{DeviceDeploymentStatusInstalling, false},
		{DeviceDeploymentStatusDownloading, false},
	}
	for _, tc := range tcs {
		if tc.finished {
			assert.True(t, IsDeviceDeploymentStatusFinished(tc.status))
		} else {
			assert.False(t, IsDeviceDeploymentStatusFinished(tc.status))
		}
	}
}
