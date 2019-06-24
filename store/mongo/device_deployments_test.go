// Copyright 2019 Northern.tech AS
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

package mongo

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/mendersoftware/go-lib-micro/identity"
	ctxstore "github.com/mendersoftware/go-lib-micro/store"
	"github.com/stretchr/testify/assert"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/utils/pointers"
)

func TestDeviceDeploymentStorageInsert(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestDeviceDeploymentStorageInsert in short mode.")
	}

	deviceDepl1, err := model.NewDeviceDeployment("30b3e62c-9ec2-4312-a7fa-cff24cc7397a", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a")
	assert.NoError(t, err)

	deviceDepl2, err := model.NewDeviceDeployment("30b3e62c-9ec2-4312-a7fa-cff24cc7397a", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a")
	assert.NoError(t, err)

	badDeviceDepl, err := model.NewDeviceDeployment("bad bad", "bad bad bad")
	assert.NoError(t, err)

	testCases := []struct {
		InputDeviceDeployment []*model.DeviceDeployment
		InputTenant           string
		OutputError           error
	}{
		{
			InputDeviceDeployment: nil,
			OutputError:           nil,
		},
		{
			InputDeviceDeployment: []*model.DeviceDeployment{nil, nil},
			OutputError:           ErrStorageInvalidDeviceDeployment,
		},
		{
			InputDeviceDeployment: []*model.DeviceDeployment{
				badDeviceDepl,
				badDeviceDepl,
			},
			OutputError: errors.New("Validating device deployment: DeploymentId: bad bad bad does not validate as uuidv4"),
		},
		{
			InputDeviceDeployment: []*model.DeviceDeployment{
				deviceDepl1,
				badDeviceDepl,
			},
			OutputError: errors.New("Validating device deployment: DeploymentId: bad bad bad does not validate as uuidv4"),
		},
		{
			InputDeviceDeployment: []*model.DeviceDeployment{
				deviceDepl1,
				deviceDepl2,
			},
			OutputError: nil,
		},
		{
			// same as previous case, but this time with tenant DB
			InputDeviceDeployment: []*model.DeviceDeployment{
				deviceDepl1,
				deviceDepl2,
			},
			InputTenant: "acme",
			OutputError: nil,
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			// Make sure we start test with empty database
			db.Wipe()

			session := db.Session()
			store := NewDataStoreMongoWithSession(session)

			ctx := context.Background()
			if testCase.InputTenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Tenant: testCase.InputTenant,
				})
			}

			err := store.InsertMany(ctx,
				testCase.InputDeviceDeployment...)

			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)

				count, err := session.DB(ctxstore.DbFromContext(ctx, DatabaseName)).
					C(CollectionDevices).
					Find(nil).Count()
				assert.NoError(t, err)
				assert.Equal(t, len(testCase.InputDeviceDeployment), count)

				if testCase.InputTenant != "" {
					// deployment was added to tenant's DB,
					// make sure it's not in default DB
					count, err := session.DB(DatabaseName).
						C(CollectionDevices).
						Find(nil).Count()
					assert.NoError(t, err)
					assert.Equal(t, 0, count)
				}
			}

			// Need to close all sessions to be able to call wipe at next test case
			session.Close()
		})
	}
}

func TestUpdateDeviceDeploymentStatus(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestUpdateDeviceDeploymentStatus in short mode.")
	}

	now := time.Now()

	deviceDeployments := []model.DeviceDeployment{}

	dds := []struct {
		did   string
		depid string
	}{
		{"456", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
		{"567", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
		{"678", "30b3e62c-9ec2-4312-a7fa-cff24cc7397d"},
		{"12345", "30b3e62c-9ec2-4312-a7fa-cff24cc7397e"},
	}

	for _, dd := range dds {
		newdd, err := model.NewDeviceDeployment(dd.did, dd.depid)
		assert.NoError(t, err)
		deviceDeployments = append(deviceDeployments, *newdd)
	}

	testCases := []struct {
		InputDeviceID         string
		InputDeploymentID     string
		InputStatus           string
		InputSubState         *string
		InputDeviceDeployment []*model.DeviceDeployment
		InputFinishTime       *time.Time
		InputTenant           string

		OutputError     error
		OutputOldStatus string
	}{
		{
			// null status
			InputDeviceID:     "123",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			OutputError:       ErrStorageInvalidInput,
			OutputOldStatus:   "",
		},
		{
			// null deployment ID
			InputDeviceID:   "234",
			InputStatus:     "",
			OutputError:     ErrStorageInvalidID,
			OutputOldStatus: "",
		},
		{
			// null device ID
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputStatus:       "notnull",
			OutputError:       ErrStorageInvalidID,
			OutputOldStatus:   "",
		},
		{
			// no deployment/device with this ID
			InputDeviceID:     "345",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputStatus:       "notnull",
			OutputError:       ErrStorageNotFound,
			OutputOldStatus:   "",
		},
		{
			InputDeviceID:     "456",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputStatus:       model.DeviceDeploymentStatusInstalling,
			InputDeviceDeployment: []*model.DeviceDeployment{
				&deviceDeployments[0],
			},
			OutputError:     nil,
			OutputOldStatus: "pending",
		},
		{
			InputDeviceID:     "567",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputStatus:       model.DeviceDeploymentStatusFailure,
			InputDeviceDeployment: []*model.DeviceDeployment{
				&deviceDeployments[1],
			},
			InputFinishTime: &now,
			OutputError:     nil,
			OutputOldStatus: "pending",
		},
		{
			InputDeviceID:     "678",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397d",
			InputStatus:       model.DeviceDeploymentStatusInstalling,
			InputDeviceDeployment: []*model.DeviceDeployment{
				&deviceDeployments[2],
			},
			InputTenant:     "acme",
			OutputOldStatus: "pending",
		},
		{
			InputDeviceID:     "12345",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397e",
			InputStatus:       model.DeviceDeploymentStatusInstalling,
			InputSubState:     pointers.StringToPointer("foobar 123"),
			InputDeviceDeployment: []*model.DeviceDeployment{
				&deviceDeployments[3],
			},
			OutputError:     nil,
			OutputOldStatus: "pending",
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			t.Logf("testing case %s %s %s %v",
				testCase.InputDeviceID, testCase.InputDeploymentID,
				testCase.InputStatus, testCase.OutputError)

			// Make sure we start test with empty database
			db.Wipe()

			session := db.Session()
			defer session.Close()

			store := NewDataStoreMongoWithSession(session)

			ctx := context.Background()
			if testCase.InputTenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Tenant: testCase.InputTenant,
				})
			}

			// deployments are created with status DeviceDeploymentStatusPending
			err := store.InsertMany(ctx, testCase.InputDeviceDeployment...)
			assert.NoError(t, err)

			old, err := store.UpdateDeviceDeploymentStatus(ctx,
				testCase.InputDeviceID, testCase.InputDeploymentID,
				model.DeviceDeploymentStatus{
					Status:     testCase.InputStatus,
					SubState:   testCase.InputSubState,
					FinishTime: testCase.InputFinishTime,
				})

			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)

				if testCase.InputTenant != "" {
					// update in tenant's DB was successful,
					// similar update in default DB should
					// fail because deployments are present
					// in tenant's DB only
					_, err := store.UpdateDeviceDeploymentStatus(context.Background(),
						testCase.InputDeviceID, testCase.InputDeploymentID,
						model.DeviceDeploymentStatus{
							Status:     testCase.InputStatus,
							FinishTime: testCase.InputFinishTime,
							SubState:   testCase.InputSubState,
						})
					t.Logf("error: %+v", err)
					assert.EqualError(t, err, ErrStorageNotFound.Error())
				}
			}

			if testCase.InputDeviceDeployment != nil {
				// these checks only make sense if there are any deployments in database
				var deployment *model.DeviceDeployment
				dep := session.DB(ctxstore.DbFromContext(ctx, DatabaseName)).C(CollectionDevices)
				query := bson.M{
					StorageKeyDeviceDeploymentDeviceId:     testCase.InputDeviceID,
					StorageKeyDeviceDeploymentDeploymentID: testCase.InputDeploymentID,
				}
				err := dep.Find(query).One(&deployment)
				assert.NoError(t, err)
				if testCase.OutputError != nil {
					// status must be unchanged in case of errors
					assert.Equal(t, model.DeviceDeploymentStatusPending, deployment.Status)
				} else {
					if !assert.NotNil(t, deployment) {
						return
					}

					assert.Equal(t, testCase.InputStatus, *deployment.Status)
					assert.Equal(t, testCase.OutputOldStatus, old)
					// verify deployment finish time
					if testCase.InputFinishTime != nil && assert.NotNil(t, deployment.Finished) {
						// mongo might have trimmed our
						// time a bit, let's check that
						// we are within a 1s range
						assert.WithinDuration(t, *testCase.InputFinishTime,
							*deployment.Finished, time.Second)
					}

					if testCase.InputSubState != nil {
						assert.Equal(t, *testCase.InputSubState, *deployment.SubState)
					}
				}
			}
		})
	}
}

func TestUpdateDeviceDeploymentLogAvailability(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestUpdateDeviceDeploymentLogAvailability in short mode.")
	}

	dd, err := model.NewDeviceDeployment("456", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a")
	assert.NoError(t, err)

	testCases := []struct {
		InputDeviceID         string
		InputDeploymentID     string
		InputLog              bool
		InputDeviceDeployment []*model.DeviceDeployment
		InputTenant           string

		OutputError error
	}{
		{
			// null deployment ID
			InputDeviceID: "234",
			OutputError:   ErrStorageInvalidID,
		},
		{
			// null device ID
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			OutputError:       ErrStorageInvalidID,
		},
		{
			// no deployment/device with this ID
			InputDeviceID:     "345",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputLog:          true,
			OutputError:       ErrStorageNotFound,
		},
		{
			InputDeviceID:     "456",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputLog:          true,
			InputDeviceDeployment: []*model.DeviceDeployment{
				dd,
			},
			OutputError: nil,
		},
		{
			InputDeviceID:     "456",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputLog:          false,
			InputDeviceDeployment: []*model.DeviceDeployment{
				dd,
			},
			InputTenant: "acme",
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			t.Logf("testing case %s %s %t %v",
				testCase.InputDeviceID, testCase.InputDeploymentID,
				testCase.InputLog, testCase.OutputError)

			// Make sure we start test with empty database
			db.Wipe()

			session := db.Session()
			store := NewDataStoreMongoWithSession(session)

			ctx := context.Background()
			if testCase.InputTenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Tenant: testCase.InputTenant,
				})
			}

			// deployments are created with status DeviceDeploymentStatusPending
			err := store.InsertMany(ctx, testCase.InputDeviceDeployment...)
			assert.NoError(t, err)

			err = store.UpdateDeviceDeploymentLogAvailability(ctx,
				testCase.InputDeviceID, testCase.InputDeploymentID, testCase.InputLog)

			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)

				if testCase.InputTenant != "" {
					// we're using tenant's DB, so acting on default DB should fail
					err := store.UpdateDeviceDeploymentLogAvailability(context.Background(),
						testCase.InputDeviceID, testCase.InputDeploymentID,
						testCase.InputLog)
					assert.EqualError(t, err, ErrStorageNotFound.Error())
				}
			}

			if testCase.InputDeviceDeployment != nil {
				var deployment *model.DeviceDeployment
				query := bson.M{
					StorageKeyDeviceDeploymentDeviceId:     testCase.InputDeviceID,
					StorageKeyDeviceDeploymentDeploymentID: testCase.InputDeploymentID,
				}
				err := session.DB(ctxstore.DbFromContext(ctx, DatabaseName)).
					C(CollectionDevices).
					Find(query).One(&deployment)

				assert.NoError(t, err)
				assert.Equal(t, testCase.InputLog, deployment.IsLogAvailable)
			}

			// Need to close all sessions to be able to call wipe at next test case
			session.Close()
		})
	}
}

func newDeviceDeploymentWithStatus(t *testing.T, deviceID string, deploymentID string, status string) *model.DeviceDeployment {
	d, err := model.NewDeviceDeployment(deviceID, deploymentID)
	assert.NoError(t, err)

	d.Status = &status
	return d
}

func TestAggregateDeviceDeploymentByStatus(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestAggregateDeviceDeploymentByStatus in short mode.")
	}

	testCases := []struct {
		InputDeploymentID     string
		InputDeviceDeployment []*model.DeviceDeployment
		InputTenant           string
		OutputError           error
		OutputStats           model.Stats
	}{
		{
			InputDeploymentID:     "ee13ea8b-a6d3-4d4c-99a6-bcfcaebc7ec3",
			InputDeviceDeployment: nil,
			OutputError:           nil,
			OutputStats:           nil,
		},
		{
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputDeviceDeployment: []*model.DeviceDeployment{
				newDeviceDeploymentWithStatus(t, "123", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					model.DeviceDeploymentStatusFailure),
				newDeviceDeploymentWithStatus(t, "234", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					model.DeviceDeploymentStatusFailure),
				newDeviceDeploymentWithStatus(t, "456", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					model.DeviceDeploymentStatusSuccess),

				// these 2 count as in progress
				newDeviceDeploymentWithStatus(t, "567", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					model.DeviceDeploymentStatusDownloading),
				newDeviceDeploymentWithStatus(t, "678", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					model.DeviceDeploymentStatusRebooting),

				newDeviceDeploymentWithStatus(t, "789", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					model.DeviceDeploymentStatusPending),
			},
			OutputError: nil,
			OutputStats: model.Stats{
				model.DeviceDeploymentStatusPending:        1,
				model.DeviceDeploymentStatusSuccess:        1,
				model.DeviceDeploymentStatusFailure:        2,
				model.DeviceDeploymentStatusRebooting:      1,
				model.DeviceDeploymentStatusDownloading:    1,
				model.DeviceDeploymentStatusInstalling:     0,
				model.DeviceDeploymentStatusNoArtifact:     0,
				model.DeviceDeploymentStatusAlreadyInst:    0,
				model.DeviceDeploymentStatusAborted:        0,
				model.DeviceDeploymentStatusDecommissioned: 0,
			},
		},
		{
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputDeviceDeployment: []*model.DeviceDeployment{
				newDeviceDeploymentWithStatus(t, "123", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					model.DeviceDeploymentStatusFailure),
				newDeviceDeploymentWithStatus(t, "456", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					model.DeviceDeploymentStatusSuccess),
			},
			InputTenant: "acme",
			OutputStats: newTestStats(model.Stats{
				model.DeviceDeploymentStatusSuccess: 1,
				model.DeviceDeploymentStatusFailure: 1,
			}),
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			t.Logf("testing case %s %v %d", testCase.InputDeploymentID, testCase.OutputError,
				len(testCase.InputDeviceDeployment))

			// Make sure we start test with empty database
			db.Wipe()

			session := db.Session()
			store := NewDataStoreMongoWithSession(session)

			ctx := context.Background()
			if testCase.InputTenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Tenant: testCase.InputTenant,
				})
			}

			err := store.InsertMany(ctx, testCase.InputDeviceDeployment...)
			assert.NoError(t, err)

			stats, err := store.AggregateDeviceDeploymentByStatus(ctx,
				testCase.InputDeploymentID)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)

				if testCase.InputTenant != "" {
					// data was inserted into tenant's DB,
					// verify that aggregates are all 0
					stats, err := store.AggregateDeviceDeploymentByStatus(context.Background(),
						testCase.InputDeploymentID)
					assert.NoError(t, err)
					assert.Equal(t, newTestStats(model.Stats{}), stats)
				}
			}

			if testCase.OutputStats != nil {
				assert.NotNil(t, stats)
				assert.Equal(t, testCase.OutputStats, stats)
			}

			// Need to close all sessions to be able to call wipe at next test case
			session.Close()
		})
	}
}

func TestGetDeviceStatusesForDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping GetDeviceStatusesForDeployment in short mode.")
	}

	input := []*model.DeviceDeployment{}

	dds := []struct {
		did   string
		depid string
	}{
		{"device0001", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
		{"device0002", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
		{"device0003", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
		{"device0004", "30b3e62c-9ec2-4312-a7fa-cff24cc7397b"},
		{"device0005", "30b3e62c-9ec2-4312-a7fa-cff24cc7397b"},
	}

	for _, dd := range dds {
		newdd, err := model.NewDeviceDeployment(dd.did, dd.depid)
		assert.NoError(t, err)
		input = append(input, newdd)
	}

	testCases := map[string]struct {
		caseId string
		tenant string

		inputDeploymentId string
		outputStatuses    []*model.DeviceDeployment
	}{
		"existing deployments 1": {
			inputDeploymentId: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			outputStatuses:    input[:3],
		},
		"existing deployments 2": {
			inputDeploymentId: "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",
			outputStatuses:    input[3:],
		},
		"nonexistent deployment": {
			inputDeploymentId: "aaaaaaaa-9ec2-4312-a7fa-cff24cc7397b",
			outputStatuses:    []*model.DeviceDeployment{},
		},
		"tenant, existing deployments": {
			inputDeploymentId: "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",
			tenant:            "acme",
			outputStatuses:    input[3:],
		},
	}

	for testCaseName, tc := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			// setup db - once for all cases
			db.Wipe()

			session := db.Session()
			store := NewDataStoreMongoWithSession(session)

			ctx := context.Background()
			if tc.tenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Tenant: tc.tenant,
				})
			}

			err := store.InsertMany(ctx, input...)
			assert.NoError(t, err)

			statuses, err := store.GetDeviceStatusesForDeployment(ctx,
				tc.inputDeploymentId)
			assert.NoError(t, err)

			assert.Equal(t, len(tc.outputStatuses), len(statuses))
			for i, out := range tc.outputStatuses {
				assert.Equal(t, out.DeviceId, statuses[i].DeviceId)
				assert.Equal(t, out.DeploymentId, statuses[i].DeploymentId)
			}

			if tc.tenant != "" {
				// deployment statuses are present in tenant's
				// DB, verify that listing from default DB
				// yields empty list
				statuses, err := store.GetDeviceStatusesForDeployment(context.Background(),
					tc.inputDeploymentId)
				assert.NoError(t, err)
				assert.Len(t, statuses, 0)
			}

			session.Close()
		})
	}
}

func TestHasDeploymentForDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping GetDeviceStatusesForDeployment in short mode.")
	}

	input := []*model.DeviceDeployment{}

	dds := []struct {
		did   string
		depid string
	}{
		{"device0001", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
		{"device0002", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
		{"device0003", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
	}

	for _, dd := range dds {
		newdd, err := model.NewDeviceDeployment(dd.did, dd.depid)
		assert.NoError(t, err)
		input = append(input, newdd)
	}

	testCases := []struct {
		deviceID     string
		deploymentID string
		tenant       string

		has bool
		err error
	}{
		{
			deviceID:     "device0001",
			deploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			has:          true,
			err:          nil,
		},
		{
			deviceID:     "device0002",
			deploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			has:          true,
			err:          nil,
		},
		{
			deviceID:     "device0003",
			deploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",
			has:          false,
		},
		{
			deviceID:     "device0004",
			deploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397c",
			has:          false,
		},
		{
			deviceID:     "device0003",
			deploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			has:          true,
			tenant:       "acme",
		},
	}

	for testCaseNumber, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			t.Logf("testing case: %v %v %v %v", tc.deviceID, tc.deploymentID, tc.has, tc.err)

			db.Wipe()

			session := db.Session()
			store := NewDataStoreMongoWithSession(session)

			ctx := context.Background()
			if tc.tenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Tenant: tc.tenant,
				})
			}

			err := store.InsertMany(ctx, input...)
			assert.NoError(t, err)

			has, err := store.HasDeploymentForDevice(ctx,
				tc.deploymentID, tc.deviceID)
			if tc.err != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.err.Error())
				assert.False(t, has)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.has, has)

				if tc.tenant != "" {
					// data was added to tenant's DB, verify
					// that there's no deployment if looking
					// in default DB
					has, err := store.HasDeploymentForDevice(context.Background(),
						tc.deploymentID, tc.deviceID)
					assert.False(t, has)
					assert.NoError(t, err)
				}
			}

			session.Close()
		})
	}
}

func TestGetDeviceDeploymentStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping GetDeviceDeploymentStatus in short mode.")
	}

	input := []*model.DeviceDeployment{}

	dds := []struct {
		did   string
		depid string
	}{
		{"device0001", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
		{"device0002", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
		{"device0003", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
	}

	for _, dd := range dds {
		newdd, err := model.NewDeviceDeployment(dd.did, dd.depid)
		assert.NoError(t, err)
		input = append(input, newdd)
	}

	testCases := map[string]struct {
		deviceID     string
		deploymentID string
		tenant       string

		status string
	}{
		"device deployment exists": {
			deviceID:     "device0001",
			deploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			status:       "pending",
		},
		"deployment not exists": {
			deviceID:     "device0003",
			deploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",
			status:       "",
		},
		"no deployment for device": {
			deviceID:     "device0004",
			deploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397c",
			status:       "",
		},
		"tenant, device deployment exists": {
			deviceID:     "device0001",
			deploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			status:       "pending",
			tenant:       "acme",
		},
	}

	for testCaseName, tc := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			t.Logf("testing case: %v %v %v", tc.deviceID, tc.deploymentID, tc.status)

			db.Wipe()

			session := db.Session()
			store := NewDataStoreMongoWithSession(session)

			ctx := context.Background()
			if tc.tenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Tenant: tc.tenant,
				})
			}

			err := store.InsertMany(ctx, input...)
			assert.NoError(t, err)

			status, err := store.GetDeviceDeploymentStatus(ctx,
				tc.deploymentID, tc.deviceID)
			assert.NoError(t, err)
			assert.Equal(t, tc.status, status)

			if tc.tenant != "" {
				// data was added to tenant's DB, trying to
				// fetch it from default DB will not fail but
				// returns empty status instead
				status, err := store.GetDeviceDeploymentStatus(context.Background(),
					tc.deploymentID, tc.deviceID)
				assert.NoError(t, err)
				assert.Equal(t, "", status)
			}

			session.Close()
		})
	}

}

func TestAbortDeviceDeployments(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestAbortDeviceDeployments in short mode.")
	}

	input := []*model.DeviceDeployment{}

	dds := []struct {
		did   string
		depid string
	}{
		{"456", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
		{"567", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
	}

	for _, dd := range dds {
		newdd, err := model.NewDeviceDeployment(dd.did, dd.depid)
		assert.NoError(t, err)
		input = append(input, newdd)
	}

	testCases := map[string]struct {
		InputDeploymentID     string
		InputDeviceDeployment []*model.DeviceDeployment

		OutputError error
	}{
		"null deployment id": {
			OutputError: ErrStorageInvalidID,
		},
		"all correct": {
			InputDeploymentID:     "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputDeviceDeployment: []*model.DeviceDeployment{},
			OutputError:           nil,
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			// Make sure we start test with empty database
			db.Wipe()

			session := db.Session()
			store := NewDataStoreMongoWithSession(session)

			err := store.InsertMany(context.Background(), testCase.InputDeviceDeployment...)
			assert.NoError(t, err)

			err = store.AbortDeviceDeployments(context.Background(), testCase.InputDeploymentID)

			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}

			if testCase.InputDeviceDeployment != nil {
				// these checks only make sense if there are any deployments in database
				var deploymentList []model.DeviceDeployment
				dep := session.DB(DatabaseName).C(CollectionDevices)
				query := bson.M{
					StorageKeyDeviceDeploymentDeploymentID: testCase.InputDeploymentID,
				}
				err := dep.Find(query).All(&deploymentList)
				assert.NoError(t, err)

				if testCase.OutputError != nil {
					for _, deployment := range deploymentList {
						// status must be unchanged in case of errors
						assert.Equal(t, model.DeviceDeploymentStatusPending,
							*deployment.Status)
					}
				} else {
					for _, deployment := range deploymentList {
						assert.Equal(t, model.DeviceDeploymentStatusAborted,
							*deployment.Status)
					}
				}
			}

			// Need to close all sessions to be able to call wipe at next test case
			session.Close()
		})
	}
}

func TestDecommissionDeviceDeployments(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestDecommissionDeviceDeployments in short mode.")
	}

	input := []*model.DeviceDeployment{}

	dds := []struct {
		did   string
		depid string
	}{
		{"foo", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
		{"bar", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"},
	}

	for _, dd := range dds {
		newdd, err := model.NewDeviceDeployment(dd.did, dd.depid)
		assert.NoError(t, err)
		input = append(input, newdd)
	}

	testCases := map[string]struct {
		InputDeviceId         string
		InputDeviceDeployment []*model.DeviceDeployment

		OutputError error
	}{
		"null device id": {
			OutputError: ErrStorageInvalidID,
		},
		"all correct": {
			InputDeviceId:         "foo",
			InputDeviceDeployment: input,
			OutputError:           nil,
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			// Make sure we start test with empty database
			db.Wipe()

			session := db.Session()
			store := NewDataStoreMongoWithSession(session)

			err := store.InsertMany(context.Background(), testCase.InputDeviceDeployment...)
			assert.NoError(t, err)

			err = store.DecommissionDeviceDeployments(context.Background(), testCase.InputDeviceId)

			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}

			if testCase.InputDeviceDeployment != nil {
				// these checks only make sense if there are any deployments in database
				var deploymentList []model.DeviceDeployment
				dep := session.DB(DatabaseName).C(CollectionDevices)
				query := bson.M{
					StorageKeyDeviceDeploymentDeviceId: testCase.InputDeviceId,
				}
				err := dep.Find(query).All(&deploymentList)
				assert.NoError(t, err)

				if testCase.OutputError != nil {
					for _, deployment := range deploymentList {
						// status must be unchanged in case of errors
						assert.Equal(t, model.DeviceDeploymentStatusPending,
							*deployment.Status)
					}
				} else {
					for _, deployment := range deploymentList {
						assert.Equal(t, model.DeviceDeploymentStatusDecommissioned,
							*deployment.Status)
					}
				}
			}

			// Need to close all sessions to be able to call wipe at next test case
			session.Close()
		})
	}
}
