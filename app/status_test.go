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

package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mendersoftware/deployments/model"
	fs_mocks "github.com/mendersoftware/deployments/s3/mocks"
	"github.com/mendersoftware/deployments/store/mocks"
)

// separate set of tests for assert if correct deployment status tracking

func TestUpdateDeviceDeploymentStatus(t *testing.T) {
	ctx := context.TODO()

	// 'downloading' -> 'installing'
	ddStatusNew := model.DeviceDeploymentStatus{
		Status: model.DeviceDeploymentStatusInstalling,
	}

	devId := "somedevice"

	depName := "foo"
	depArtifact := "bar"
	fakeDeployment, err := model.NewDeploymentFromConstructor(
		&model.DeploymentConstructor{
			Name:         &depName,
			ArtifactName: &depArtifact,
			Devices:      []string{"baz"},
		},
	)
	assert.NoError(t, err)

	fakeDeviceDeployment, err := model.NewDeviceDeployment(
		devId, *fakeDeployment.Id)
	status := model.DeviceDeploymentStatusDownloading
	fakeDeviceDeployment.Status = &status

	fs := &fs_mocks.FileStorage{}
	db := mocks.DataStore{}

	db.On("GetDeviceDeployment", ctx,
		*fakeDeployment.Id, devId).Return(
		fakeDeviceDeployment, nil)

	db.On("UpdateDeviceDeploymentStatus", ctx,
		devId,
		*fakeDeployment.Id,
		mock.MatchedBy(func(ddStatus model.DeviceDeploymentStatus) bool {
			assert.Equal(t, model.DeviceDeploymentStatusInstalling, ddStatus.Status)

			return true
		})).Return(model.DeviceDeploymentStatusDownloading, nil)

	db.On("UpdateStatsInc", ctx,
		*fakeDeployment.Id,
		model.DeviceDeploymentStatusDownloading,
		model.DeviceDeploymentStatusInstalling).Return(nil)

	// fake updated stats
	fakeDeployment.Stats[model.DeviceDeploymentStatusInstalling] = 1

	db.On("FindDeploymentByID", ctx, *fakeDeployment.Id).Return(
		fakeDeployment, nil)

	db.On("SetDeploymentStatus", ctx,
		*fakeDeployment.Id,
		"inprogress",
		mock.AnythingOfType("time.Time")).Return(nil)

	ds := NewDeployments(&db, fs, "")

	err = ds.UpdateDeviceDeploymentStatus(ctx, *fakeDeployment.Id, *fakeDeviceDeployment.DeviceId, ddStatusNew)
	assert.NoError(t, err)

}

func TestGetDeploymentForDeviceWithCurrent(t *testing.T) {
	ctx := context.TODO()

	// for simplicity - alreadyinstalled case
	devId := "somedevice"
	devType := "baz"

	depName := "foo"
	depArtifact := "bar"

	installed := model.InstalledDeviceDeployment{
		ArtifactName: depArtifact,
		DeviceType:   devType,
	}

	fakeDeployment, err := model.NewDeploymentFromConstructor(
		&model.DeploymentConstructor{
			Name:         &depName,
			ArtifactName: &depArtifact,
			Devices:      []string{devType},
		},
	)
	assert.NoError(t, err)

	fakeDeviceDeployment, err := model.NewDeviceDeployment(
		devId, *fakeDeployment.Id)
	status := model.DeviceDeploymentStatusPending
	fakeDeviceDeployment.Status = &status

	fs := &fs_mocks.FileStorage{}
	db := mocks.DataStore{}

	db.On("FindOldestDeploymentForDeviceIDWithStatuses", ctx, devId,
		model.ActiveDeploymentStatuses()).Return(
		fakeDeviceDeployment, nil)

	db.On("FindDeploymentByID", ctx, *fakeDeployment.Id).Return(
		fakeDeployment, nil).Once()

	db.On("DeviceCountByDeployment", ctx, *fakeDeployment.Id).Return(2, nil)
	db.On("GetDeviceDeployment", ctx,
		*fakeDeployment.Id, *fakeDeviceDeployment.DeviceId).Return(
		fakeDeviceDeployment, nil)

	db.On("IncrementDeviceDeploymentAttempts", ctx,
		*fakeDeviceDeployment.Id, uint(1)).Return(nil)

	db.On("UpdateDeviceDeploymentStatus", ctx,
		*fakeDeviceDeployment.DeviceId,
		*fakeDeployment.Id,

		mock.MatchedBy(func(ddStatus model.DeviceDeploymentStatus) bool {
			assert.Equal(t, model.DeviceDeploymentStatusAlreadyInst, ddStatus.Status)

			return true
		})).Return(model.DeviceDeploymentStatusPending, nil)

	db.On("UpdateStatsInc", ctx,
		*fakeDeployment.Id,
		model.DeviceDeploymentStatusPending,
		model.DeviceDeploymentStatusAlreadyInst).Return(nil)

	// fake updated stats
	fakeDeployment.Stats[model.DeviceDeploymentStatusNoArtifact] = 1
	db.On("FindDeploymentByID", ctx, *fakeDeployment.Id).Return(
		fakeDeployment, nil)

	db.On("SetDeploymentStatus", ctx,
		*fakeDeployment.Id,
		"finished",
		mock.AnythingOfType("time.Time")).Return(nil)

	ds := NewDeployments(&db, fs, "")

	_, err = ds.GetDeploymentForDeviceWithCurrent(ctx, devId, &installed)
	assert.NoError(t, err)
}

func TestAbortDeployment(t *testing.T) {
	ctx := context.TODO()

	depId := "foo"
	stats := model.NewDeviceDeploymentStats()

	db := mocks.DataStore{}
	db.On("AbortDeviceDeployments", ctx, depId).Return(nil)
	stats[model.DeviceDeploymentStatusAborted] = 10

	db.On("AggregateDeviceDeploymentByStatus", ctx, depId).Return(stats, nil)

	db.On("UpdateStats", ctx, depId, stats).Return(nil)

	db.On("SetDeploymentStatus", ctx,
		depId,
		"finished",
		mock.AnythingOfType("time.Time")).Return(nil)

	ds := NewDeployments(&db, nil, "")

	err := ds.AbortDeployment(ctx, "foo")
	assert.NoError(t, err)
}

func TestDecommission(t *testing.T) {
	ctx := context.TODO()

	devId := "foo"
	depId := "bar"

	db := mocks.DataStore{}
	db.On("DecommissionDeviceDeployments", ctx, devId).Return(nil)

	dds := []model.DeviceDeployment{
		model.DeviceDeployment{
			DeploymentId: &depId,
			DeviceId:     &devId,
		},
	}
	db.On("FindAllDeploymentsForDeviceIDWithStatuses", ctx,
		devId,
		[]string{model.DeviceDeploymentStatusDecommissioned}).Return(dds, nil)

	stats := model.NewDeviceDeploymentStats()
	stats[model.DeviceDeploymentStatusDecommissioned] = 1

	db.On("AggregateDeviceDeploymentByStatus", ctx, depId).Return(stats, nil)

	db.On("UpdateStats", ctx, depId, stats).Return(nil)

	db.On("SetDeploymentStatus", ctx,
		depId,
		"finished",
		mock.AnythingOfType("time.Time")).Return(nil)

	ds := NewDeployments(&db, nil, "")

	err := ds.DecommissionDevice(ctx, devId)
	assert.NoError(t, err)
}
