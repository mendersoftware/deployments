// Copyright 2020 Northern.tech AS
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

package model

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/mendersoftware/deployments/utils/pointers"
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

		dep := &DeploymentConstructor{}
		dep.Name = test.InputName
		dep.ArtifactName = test.InputArtifactName
		dep.Devices = test.InputDevices

		err := dep.Validate("")

		if !test.IsValid {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}

}

func TestNewDeploymentFromConstructor(t *testing.T) {

	t.Parallel()

	dep, err := NewDeploymentFromConstructor(nil)
	assert.NoError(t, err)
	assert.NotNil(t, dep)

	con := &DeploymentConstructor{}

	dep, err = NewDeploymentFromConstructor(con)
	assert.NoError(t, err)

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
		{
			InputName:         StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputArtifactName: StringToPointer("f826484e-1157-4109-af21-304e6d711560"),
			InputDevices:      []string{},
			IsValid:           false,
		},
	}

	for _, test := range testCases {

		pub := &DeploymentConstructor{}
		pub.Name = test.InputName
		pub.ArtifactName = test.InputArtifactName
		pub.Devices = test.InputDevices

		dep, err := NewDeploymentFromConstructor(pub)
		assert.NoError(t, err)

		err = dep.Validate()

		if !test.IsValid {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}

}

func TestDeploymentMarshalJSON(t *testing.T) {

	t.Parallel()

	dep, err := NewDeployment()
	assert.NoError(t, err)
	dep.Name = StringToPointer("Region: NYC")
	dep.ArtifactName = StringToPointer("App 123")
	dep.Devices = []string{"Device 123"}
	dep.Id = StringToPointer("14ddec54-30be-49bf-aa6b-97ce271d71f5")
	deviceCount := 1337
	dep.DeviceCount = &deviceCount
	dep.Status = DeploymentStatusInProgress

	j, err := dep.MarshalJSON()
	assert.NoError(t, err)

	// date format may be slightly different on different platforms
	expectedJSON := `
    {
        "name": "Region: NYC",
        "artifact_name": "App 123",
        "created":"` + dep.Created.Format(time.RFC3339Nano) + `",
        "device_count": 1337,
        "id":"14ddec54-30be-49bf-aa6b-97ce271d71f5",
        "status": "inprogress"
    }`

	assert.JSONEq(t, expectedJSON, string(j))
}

func TestDeploymentIs(t *testing.T) {
	d, err := NewDeployment()
	assert.NoError(t, err)
	d.MaxDevices = 1

	assert.False(t, d.IsNotPending())
	assert.False(t, d.IsFinished())

	// check all active statuses
	active := []string{
		DeviceDeploymentStatusRebooting,
		DeviceDeploymentStatusInstalling,
		DeviceDeploymentStatusDownloading,
	}
	for _, as := range active {
		t.Logf("checking in-progress deployment stat %s", as)
		d.Stats = NewDeviceDeploymentStats()
		d.Stats[as] = 1
		assert.True(t, d.IsNotPending())
		assert.False(t, d.IsFinished())
	}

	finished := []string{
		DeviceDeploymentStatusSuccess,
		DeviceDeploymentStatusFailure,
		DeviceDeploymentStatusNoArtifact,
		DeviceDeploymentStatusAlreadyInst,
		DeviceDeploymentStatusAborted,
	}
	for _, as := range finished {
		t.Logf("checking finished deployment stat %s", as)
		d.Stats = NewDeviceDeploymentStats()
		d.Stats[as] = 1
		assert.True(t, d.IsFinished())
		assert.True(t, d.IsNotPending())
	}

	pending := []string{
		DeviceDeploymentStatusPending,
	}
	for _, as := range pending {
		t.Logf("checking pending deployment stat %s", as)
		d.Stats = NewDeviceDeploymentStats()
		d.Stats[as] = 1
		assert.False(t, d.IsFinished())
		assert.False(t, d.IsNotPending())
	}
}

func TestDeploymentGetStatus(t *testing.T) {

	tests := map[string]struct {
		Stats        map[string]int
		OutputStatus string
	}{
		"Single NoArtifact": {
			Stats: map[string]int{
				DeviceDeploymentStatusNoArtifact: 1,
			},
			OutputStatus: "finished",
		},
		"Single Success": {
			Stats: map[string]int{
				DeviceDeploymentStatusSuccess: 1,
			},
			OutputStatus: "finished",
		},
		"Success + NoArtifact": {
			Stats: map[string]int{
				DeviceDeploymentStatusSuccess:    1,
				DeviceDeploymentStatusNoArtifact: 1,
			},
			OutputStatus: "finished",
		},
		"Failed + NoArtifact": {
			Stats: map[string]int{
				DeviceDeploymentStatusFailure:    1,
				DeviceDeploymentStatusNoArtifact: 1,
			},
			OutputStatus: "finished",
		},
		"Failed + AlreadyInst": {
			Stats: map[string]int{
				DeviceDeploymentStatusFailure:     1,
				DeviceDeploymentStatusAlreadyInst: 1,
			},
			OutputStatus: "finished",
		},
		"Failed + Aborted": {
			Stats: map[string]int{
				DeviceDeploymentStatusFailure: 1,
				DeviceDeploymentStatusAborted: 1,
			},
			OutputStatus: "finished",
		},
		"Rebooting + NoArtifact": {
			Stats: map[string]int{
				DeviceDeploymentStatusRebooting:  1,
				DeviceDeploymentStatusNoArtifact: 1,
			},
			OutputStatus: "inprogress",
		},
		"Rebooting + Installing": {
			Stats: map[string]int{
				DeviceDeploymentStatusRebooting:  1,
				DeviceDeploymentStatusInstalling: 1,
			},
			OutputStatus: "inprogress",
		},
		"Rebooting + Pending": {
			Stats: map[string]int{
				DeviceDeploymentStatusRebooting: 1,
				DeviceDeploymentStatusPending:   1,
			},
			OutputStatus: "inprogress",
		},
		"Pending": {
			Stats: map[string]int{
				DeviceDeploymentStatusPending: 1,
			},
			OutputStatus: "pending",
		},
		"Empty": {
			OutputStatus: "finished",
		},
		//verify we count 'already-installed' towards 'inprogress'
		"pending + already-installed": {
			Stats: map[string]int{
				DeviceDeploymentStatusPending:     1,
				DeviceDeploymentStatusAlreadyInst: 1,
			},
			OutputStatus: "inprogress",
		},
		//verify we count 'already-installed' towards 'finished'
		"already-installed + finished": {
			Stats: map[string]int{
				DeviceDeploymentStatusPending:     0,
				DeviceDeploymentStatusAlreadyInst: 1,
			},
			OutputStatus: "finished",
		},
	}

	for name, test := range tests {

		t.Log(name)

		dep, err := NewDeployment()
		assert.NoError(t, err)

		dep.Stats = test.Stats
		for _, n := range dep.Stats {
			dep.MaxDevices += n
		}

		assert.Equal(t, test.OutputStatus, dep.GetStatus())
	}

}

func TestFuzzyGetStatus(t *testing.T) {

	rand := func(min int, max int) int {
		rand.Seed(time.Now().UTC().UnixNano())
		return min + rand.Intn(max-min)
	}

	max := 3

	for i := 0; i < 1000; i++ {
		dep, err := NewDeployment()
		assert.NoError(t, err)

		dep.Stats[DeviceDeploymentStatusDownloading] = rand(0, max)
		dep.Stats[DeviceDeploymentStatusInstalling] = rand(0, max)
		dep.Stats[DeviceDeploymentStatusRebooting] = rand(0, max)
		dep.Stats[DeviceDeploymentStatusPending] = rand(0, max)
		dep.Stats[DeviceDeploymentStatusSuccess] = rand(0, max)
		dep.Stats[DeviceDeploymentStatusFailure] = rand(0, max)
		dep.Stats[DeviceDeploymentStatusNoArtifact] = rand(0, max)
		dep.Stats[DeviceDeploymentStatusAlreadyInst] = rand(0, max)
		dep.Stats[DeviceDeploymentStatusAborted] = rand(0, max)
		dep.Stats[DeviceDeploymentStatusDecommissioned] = rand(0, max)

		pending := 0
		inprogress := 0
		finished := 0

		if dep.GetStatus() == "pending" {
			pending++
		}

		if dep.GetStatus() == "finished" {
			finished++
		}

		if dep.GetStatus() == "inprogress" {
			inprogress++
		}

		exp_stats := pending + inprogress + finished
		assert.Equal(t, 1, exp_stats, dep.Stats)
	}
}
