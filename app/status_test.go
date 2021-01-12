// Copyright 2021 Northern.tech AS
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
	"errors"
	"testing"
	"time"

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
	ddStatusNew := model.DeviceDeploymentState{
		Status: model.DeviceDeploymentStatusInstalling,
	}

	devId := "somedevice"

	depName := "foo"
	depArtifact := "bar"
	fakeDeployment, err := model.NewDeploymentFromConstructor(
		&model.DeploymentConstructor{
			Name:         depName,
			ArtifactName: depArtifact,
			Devices:      []string{"baz"},
		},
	)
	fakeDeployment.MaxDevices = 1
	assert.NoError(t, err)

	fakeDeviceDeployment := model.NewDeviceDeployment(
		devId, fakeDeployment.Id)
	fakeDeviceDeployment.Status = model.DeviceDeploymentStatusDownloading

	fs := &fs_mocks.FileStorage{}
	db := mocks.DataStore{}

	db.On("GetDeviceDeployment", ctx,
		fakeDeployment.Id, devId).Return(
		fakeDeviceDeployment, nil)

	db.On("UpdateDeviceDeploymentStatus", ctx,
		devId,
		fakeDeployment.Id,
		mock.MatchedBy(func(ddStatus model.DeviceDeploymentState) bool {
			assert.Equal(t, model.DeviceDeploymentStatusInstalling, ddStatus.Status)

			return true
		})).Return(model.DeviceDeploymentStatusDownloading, nil)

	db.On("UpdateStatsInc", ctx,
		fakeDeployment.Id,
		model.DeviceDeploymentStatusDownloading,
		model.DeviceDeploymentStatusInstalling).Return(nil)

	// fake updated stats
	fakeDeployment.Stats[model.DeviceDeploymentStatusInstalling] = 1

	db.On("FindDeploymentByID", ctx, fakeDeployment.Id).Return(
		fakeDeployment, nil)

	db.On("SetDeploymentStatus", ctx,
		fakeDeployment.Id,
		model.DeploymentStatusInProgress,
		mock.AnythingOfType("time.Time")).Return(nil)

	ds := NewDeployments(&db, fs, "")

	err = ds.UpdateDeviceDeploymentStatus(ctx, fakeDeployment.Id, fakeDeviceDeployment.DeviceId, ddStatusNew)
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
			Name:         depName,
			ArtifactName: depArtifact,
			Devices:      []string{devType},
		},
	)
	fakeDeployment.MaxDevices = 1
	assert.NoError(t, err)

	fakeDeviceDeployment := model.NewDeviceDeployment(
		devId, fakeDeployment.Id)
	fakeDeviceDeployment.Status = model.DeviceDeploymentStatusPending

	fs := &fs_mocks.FileStorage{}
	db := mocks.DataStore{}

	call := db.On("FindOldestDeploymentForDeviceIDWithStatuses", ctx, devId).Return(
		fakeDeviceDeployment, nil)
	// Add variadic arguments
	for _, status := range model.ActiveDeploymentStatuses() {
		call.Arguments = append(call.Arguments, interface{}(status))
	}

	db.On("FindDeploymentByID", ctx, fakeDeployment.Id).Return(
		fakeDeployment, nil).Once()

	db.On("DeviceCountByDeployment", ctx, fakeDeployment.Id).Return(2, nil)
	db.On("GetDeviceDeployment", ctx,
		fakeDeployment.Id, fakeDeviceDeployment.DeviceId).Return(
		fakeDeviceDeployment, nil)

	db.On("IncrementDeviceDeploymentAttempts", ctx,
		fakeDeviceDeployment.Id, uint(1)).Return(nil)

	db.On("UpdateDeviceDeploymentStatus", ctx,
		fakeDeviceDeployment.DeviceId,
		fakeDeployment.Id,

		mock.MatchedBy(func(ddStatus model.DeviceDeploymentState) bool {
			assert.Equal(t, model.DeviceDeploymentStatusAlreadyInst, ddStatus.Status)

			return true
		})).Return(model.DeviceDeploymentStatusPending, nil)

	db.On("UpdateStatsInc", ctx,
		fakeDeployment.Id,
		model.DeviceDeploymentStatusPending,
		model.DeviceDeploymentStatusAlreadyInst).Return(nil)

	// fake updated stats
	fakeDeployment.Stats[model.DeviceDeploymentStatusNoArtifact] = 1
	db.On("FindDeploymentByID", ctx, fakeDeployment.Id).Return(
		fakeDeployment, nil)

	db.On("SetDeploymentStatus", ctx,
		fakeDeployment.Id,
		model.DeploymentStatusFinished,
		mock.AnythingOfType("time.Time")).Return(nil)

	ds := NewDeployments(&db, fs, "")

	_, err = ds.GetDeploymentForDeviceWithCurrent(ctx, devId, &installed)
	assert.NoError(t, err)
}

func TestAbortDeployment(t *testing.T) {
	ctx := context.TODO()

	depId := "foo"
	stats := model.NewDeviceDeploymentStats()
	depName := "foo"
	depArtifact := "bar"
	fakeDeployment, err := model.NewDeploymentFromConstructor(
		&model.DeploymentConstructor{
			Name:         depName,
			ArtifactName: depArtifact,
			Devices:      []string{"baz"},
		},
	)
	fakeDeployment.MaxDevices = 1
	fakeDeployment.Stats = stats
	fakeDeployment.Id = depId
	assert.NoError(t, err)

	db := mocks.DataStore{}
	db.On("AbortDeviceDeployments", ctx, depId).Return(nil)
	stats[model.DeviceDeploymentStatusAborted] = 10

	db.On("AggregateDeviceDeploymentByStatus", ctx, depId).Return(stats, nil)

	db.On("UpdateStats", ctx, depId, stats).Return(nil)

	db.On("SetDeploymentStatus", ctx,
		depId,
		model.DeploymentStatusFinished,
		mock.AnythingOfType("time.Time")).Return(nil)

	ds := NewDeployments(&db, nil, "")

	err = ds.AbortDeployment(ctx, "foo")
	assert.NoError(t, err)
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func intPtr(i int) *int {
	return &i
}

func TestDecommission(t *testing.T) {
	testCases := map[string]struct {
		inputDeviceId       string
		inputDeploymentId   string
		inputDeploymentName string
		inputArtifactName   string
		inputMaxDevices     int
		inputStats          model.Stats
		inputDevices        []string

		deviceDeployments                                     []model.DeviceDeployment
		findOldestDeploymentForDeviceIDWithStatusesDeployment *model.DeviceDeployment
		findOldestDeploymentForDeviceIDWithStatusesError      error
		getDeviceDeploymentDeployment                         *model.DeviceDeployment
		getDeviceDeploymentError                              error
		updateDeviceDeploymentStatusStatus                    model.DeviceDeploymentStatus
		updateDeviceDeploymentStatusError                     error
		findLatestDeploymentForDeviceIDWithStatusesDeployment *model.DeviceDeployment
		findLatestDeploymentForDeviceIDWithStatusesError      error
		findNewerActiveDeploymentsDeployments                 []*model.Deployment
		findNewerActiveDeploymentsError                       error
		findDeploymentByIDDeployment                          *model.Deployment
		findDeploymentByIDError                               error
		insertDeviceDeploymentError                           error
		updateStatsIncError                                   error
		setDeploymentStatusError                              error

		outputError error
	}{
		"ok": {
			inputDeviceId:       "foo",
			inputDeploymentId:   "bar",
			inputDeploymentName: "foo",
			inputDevices:        []string{"baz"},

			findOldestDeploymentForDeviceIDWithStatusesDeployment: &model.DeviceDeployment{
				Id:           "bar",
				DeploymentId: "bar",
				Status:       model.DeviceDeploymentStatusDownloading,
			},
			getDeviceDeploymentDeployment: &model.DeviceDeployment{
				Id:           "bar",
				DeploymentId: "bar",
				Status:       model.DeviceDeploymentStatusDownloading,
			},
			updateDeviceDeploymentStatusStatus: model.DeviceDeploymentStatusDownloading,
			findDeploymentByIDDeployment: &model.Deployment{
				Id:         "bar",
				MaxDevices: 1,
				Stats:      model.Stats{"decommissioned": 1},
			},
		},
		"ok 1": {
			findLatestDeploymentForDeviceIDWithStatusesDeployment: &model.DeviceDeployment{
				Id:           "bar",
				DeploymentId: "bar",
				Status:       model.DeviceDeploymentStatusSuccess,
				Created:      timePtr(time.Now()),
			},
		},
		"ok 2": {},
		"ok 3": {
			findNewerActiveDeploymentsDeployments: []*model.Deployment{
				{},
			},
		},
		"ok 4": {
			inputDeviceId:     "foo",
			inputDeploymentId: "foo",
			findNewerActiveDeploymentsDeployments: []*model.Deployment{
				{
					DeviceList:  []string{"foo"},
					Id:          "foo",
					Created:     timePtr(time.Now()),
					DeviceCount: intPtr(0),
					MaxDevices:  1,
					Stats:       model.Stats{},
				},
			},
		},
		"ok, pending": {
			inputDeviceId:     "foo",
			inputDeploymentId: "pending",
			findNewerActiveDeploymentsDeployments: []*model.Deployment{
				{
					DeviceList:  []string{"foo"},
					Id:          "pending",
					Created:     timePtr(time.Now()),
					DeviceCount: intPtr(0),
					MaxDevices:  2,
					Stats:       model.Stats{},
				},
			},
		},
		"FindOldestDeploymentForDeviceIDWithStatuses error": {
			inputDeviceId:       "foo",
			inputDeploymentId:   "bar",
			inputDeploymentName: "foo",
			inputDevices:        []string{"baz"},

			findOldestDeploymentForDeviceIDWithStatusesError: errors.New("foo"),

			outputError: errors.New("Searching for active deployment for the device: foo"),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.TODO()
			db := mocks.DataStore{}

			call := db.On("FindOldestDeploymentForDeviceIDWithStatuses",
				ctx, tc.inputDeviceId).
				Return(
					tc.findOldestDeploymentForDeviceIDWithStatusesDeployment,
					tc.findOldestDeploymentForDeviceIDWithStatusesError,
				)
			// Add variadic arguments
			for _, status := range model.ActiveDeploymentStatuses() {
				call.Arguments = append(call.Arguments, status)
			}

			db.On("GetDeviceDeployment", ctx, tc.inputDeploymentId,
				tc.inputDeviceId).Return(
				tc.getDeviceDeploymentDeployment, tc.getDeviceDeploymentError)

			db.On("UpdateDeviceDeploymentStatus", ctx, tc.inputDeviceId,
				tc.inputDeploymentId, mock.AnythingOfType("model.DeviceDeploymentState")).Return(
				tc.updateDeviceDeploymentStatusStatus, tc.updateDeviceDeploymentStatusError)

			call = db.On("FindLatestDeploymentForDeviceIDWithStatuses",
				ctx, tc.inputDeviceId,
			).Return(
				tc.findLatestDeploymentForDeviceIDWithStatusesDeployment,
				tc.findLatestDeploymentForDeviceIDWithStatusesError,
			)
			// Add variadic arguments
			for _, status := range model.InactiveDeploymentStatuses() {
				call.Arguments = append(call.Arguments, status)
			}

			db.On("FindNewerActiveDeployments", ctx, mock.AnythingOfType("*time.Time"),
				0, 100).Return(
				tc.findNewerActiveDeploymentsDeployments, tc.findNewerActiveDeploymentsError)
			db.On("FindNewerActiveDeployments", ctx, mock.AnythingOfType("*time.Time"),
				100, 100).Return(nil, nil)
			db.On("InsertDeviceDeployment", ctx, mock.AnythingOfType("*model.DeviceDeployment")).Return(
				tc.insertDeviceDeploymentError)

			db.On("FindDeploymentByID", ctx, tc.inputDeploymentId).Return(
				tc.findDeploymentByIDDeployment, tc.findDeploymentByIDError)

			db.On("UpdateStatsInc", ctx, tc.inputDeploymentId,
				tc.updateDeviceDeploymentStatusStatus,
				model.DeviceDeploymentStatusDecommissioned).Return(tc.updateStatsIncError)

			db.On("SetDeploymentStatus", ctx,
				tc.inputDeploymentId,
				model.DeploymentStatusFinished,
				mock.AnythingOfType("time.Time")).
				Return(tc.setDeploymentStatusError).
				Once()
			db.On("SetDeploymentStatus", ctx,
				"pending",
				model.DeploymentStatusPending,
				mock.AnythingOfType("time.Time")).
				Return(tc.setDeploymentStatusError).
				Once()

			ds := NewDeployments(&db, nil, "")

			err := ds.DecommissionDevice(ctx, tc.inputDeviceId)
			if tc.outputError != nil {
				assert.EqualError(t, err, tc.outputError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
