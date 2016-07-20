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

func TestDeploymentConstructorValidate(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputName         *string
		InputArtifactName *string
		InputDevices      []string
		IsValid           bool
	}{
		{
			InputName:         nil,
			InputArtifactName: nil,
			InputDevices:      nil,
			IsValid:           false,
		},
		{
			InputName:         StringToPointer("something"),
			InputArtifactName: nil,
			InputDevices:      nil,
			IsValid:           false,
		},
		{
			InputName:         StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputArtifactName: nil,
			InputDevices:      nil,
			IsValid:           false,
		},
		{
			InputName:         StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputArtifactName: StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDevices:      nil,
			IsValid:           false,
		},
		{
			InputName:         StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputArtifactName: StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDevices:      []string{},
			IsValid:           false,
		},
		{
			InputName:         StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputArtifactName: StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDevices:      []string{""},
			IsValid:           false,
		},
		{
			InputName:         StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputArtifactName: StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDevices:      []string{"lala"},
			IsValid:           true,
		},
		{
			InputName:         StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputArtifactName: StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDevices:      []string{"f826484e-1157-4109-af21-304e6d711560"},
			IsValid:           true,
		},
	}

	for _, test := range testCases {

		dep := NewDeploymentConstructor()
		dep.Name = test.InputName
		dep.ArtifactName = test.InputArtifactName
		dep.Devices = test.InputDevices

		err := dep.Validate()

		if !test.IsValid {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}

}

func TestNewDeploymentFromConstructor(t *testing.T) {

	t.Parallel()

	assert.NotNil(t, NewDeploymentFromConstructor(nil))

	con := NewDeploymentConstructor()
	dep := NewDeploymentFromConstructor(con)
	assert.NotNil(t, dep)
	assert.Equal(t, con, dep.DeploymentConstructor)
}

func TestDeploymentValidate(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputName         *string
		InputArtifactName *string
		InputDevices      []string
		IsValid           bool
	}{
		{
			InputName:         nil,
			InputArtifactName: nil,
			InputDevices:      nil,
			IsValid:           false,
		},
		{
			InputName:         StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputArtifactName: StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDevices:      []string{"f826484e-1157-4109-af21-304e6d711560"},
			IsValid:           true,
		},
	}

	for _, test := range testCases {

		pub := NewDeploymentConstructor()
		pub.Name = test.InputName
		pub.ArtifactName = test.InputArtifactName
		pub.Devices = test.InputDevices

		dep := NewDeploymentFromConstructor(pub)

		err := dep.Validate()

		if !test.IsValid {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}

}

func TestDeploymentMarshalJSON(t *testing.T) {

	t.Parallel()

	dep := NewDeployment()
	dep.Name = StringToPointer("Region: NYC")
	dep.ArtifactName = StringToPointer("App 123")
	dep.Devices = []string{"Device 123"}
	dep.Id = StringToPointer("14ddec54-30be-49bf-aa6b-97ce271d71f5")

	j, err := dep.MarshalJSON()
	assert.NoError(t, err)

	// date format may be slightly different on different platforms
	expectedJSON := `
    {
        "name": "Region: NYC", 
        "artifact_name": "App 123", 
        "created":"` + dep.Created.Format(time.RFC3339Nano) + `", 
        "id":"14ddec54-30be-49bf-aa6b-97ce271d71f5"
    }`

	assert.JSONEq(t, expectedJSON, string(j))
}
