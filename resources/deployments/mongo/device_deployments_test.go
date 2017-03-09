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

package mongo_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/mendersoftware/deployments/resources/deployments"
	. "github.com/mendersoftware/deployments/resources/deployments/mongo"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

func TestDeviceDeploymentStorageInsert(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestDeviceDeploymentStorageInsert in short mode.")
	}

	testCases := []struct {
		InputDeviceDeployment []*deployments.DeviceDeployment
		OutputError           error
	}{
		{
			InputDeviceDeployment: nil,
			OutputError:           nil,
		},
		{
			InputDeviceDeployment: []*deployments.DeviceDeployment{nil, nil},
			OutputError:           ErrStorageInvalidDeviceDeployment,
		},
		{
			InputDeviceDeployment: []*deployments.DeviceDeployment{
				deployments.NewDeviceDeployment("bad bad", "bad bad bad"),
				deployments.NewDeviceDeployment("bad bad", "bad bad bad"),
			},
			OutputError: errors.New("Validating device deployment: DeploymentId: bad bad bad does not validate as uuidv4;"),
		},
		{
			InputDeviceDeployment: []*deployments.DeviceDeployment{
				deployments.NewDeviceDeployment("30b3e62c-9ec2-4312-a7fa-cff24cc7397a", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
				deployments.NewDeviceDeployment("bad bad", "bad bad bad"),
			},
			OutputError: errors.New("Validating device deployment: DeploymentId: bad bad bad does not validate as uuidv4;"),
		},
		{
			InputDeviceDeployment: []*deployments.DeviceDeployment{
				deployments.NewDeviceDeployment("30b3e62c-9ec2-4312-a7fa-cff24cc7397a", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
				deployments.NewDeviceDeployment("30b3e62c-9ec2-4312-a7fa-cff24cc7397a", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
			},
			OutputError: nil,
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			// Make sure we start test with empty database
			db.Wipe()

			session := db.Session()
			store := NewDeviceDeploymentsStorage(session)

			err := store.InsertMany(testCase.InputDeviceDeployment...)

			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)

				dep := session.DB(DatabaseName).C(CollectionDevices)
				count, err := dep.Find(nil).Count()
				assert.NoError(t, err)
				assert.Equal(t, len(testCase.InputDeviceDeployment), count)
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

	testCases := []struct {
		InputDeviceID         string
		InputDeploymentID     string
		InputStatus           string
		InputDeviceDeployment []*deployments.DeviceDeployment
		InputFinishTime       *time.Time

		OutputError     error
		OutputOldStatus string
	}{
		{
			// null status
			InputDeviceID:     "123",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			OutputError:       ErrStorageInvalidID,
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
			OutputError:       errors.New("not found"),
			OutputOldStatus:   "",
		},
		{
			InputDeviceID:     "456",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputStatus:       deployments.DeviceDeploymentStatusInstalling,
			InputDeviceDeployment: []*deployments.DeviceDeployment{
				deployments.NewDeviceDeployment("456", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
			},
			OutputError:     nil,
			OutputOldStatus: "pending",
		},
		{
			InputDeviceID:     "567",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputStatus:       deployments.DeviceDeploymentStatusFailure,
			InputDeviceDeployment: []*deployments.DeviceDeployment{
				deployments.NewDeviceDeployment("567", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
			},
			InputFinishTime: &now,
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
			store := NewDeviceDeploymentsStorage(session)

			// deployments are created with status DeviceDeploymentStatusPending
			err := store.InsertMany(testCase.InputDeviceDeployment...)
			assert.NoError(t, err)

			old, err := store.UpdateDeviceDeploymentStatus(testCase.InputDeviceID,
				testCase.InputDeploymentID, testCase.InputStatus, testCase.InputFinishTime)

			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}

			if testCase.InputDeviceDeployment != nil {
				// these checks only make sense if there are any deployments in database
				var deployment *deployments.DeviceDeployment
				dep := session.DB(DatabaseName).C(CollectionDevices)
				query := bson.M{
					StorageKeyDeviceDeploymentDeviceId:     testCase.InputDeviceID,
					StorageKeyDeviceDeploymentDeploymentID: testCase.InputDeploymentID,
				}
				err := dep.Find(query).One(&deployment)
				assert.NoError(t, err)

				if testCase.OutputError != nil {
					// status must be unchanged in case of errors
					assert.Equal(t, deployments.DeviceDeploymentStatusPending, deployment.Status)
				} else {
					assert.Equal(t, testCase.InputStatus, *deployment.Status)
					assert.Equal(t, testCase.OutputOldStatus, old)
					// verify deployment finish time
					if testCase.InputFinishTime != nil && assert.NotNil(t, deployment.Finished) {
						// mongo might have trimmed our time a bit, let's check that we are within a 1s range
						assert.WithinDuration(t, *testCase.InputFinishTime, *deployment.Finished, time.Second)
					}
				}
			}

			// Need to close all sessions to be able to call wipe at next test case
			session.Close()
		})
	}
}

func TestUpdateDeviceDeploymentLogAvailability(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestUpdateDeviceDeploymentLogAvailability in short mode.")
	}

	testCases := []struct {
		InputDeviceID         string
		InputDeploymentID     string
		InputLog              bool
		InputDeviceDeployment []*deployments.DeviceDeployment

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
			OutputError:       errors.New("not found"),
		},
		{
			InputDeviceID:     "456",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputLog:          true,
			InputDeviceDeployment: []*deployments.DeviceDeployment{
				deployments.NewDeviceDeployment("456", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
			},
			OutputError: nil,
		},
		{
			InputDeviceID:     "456",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputLog:          false,
			InputDeviceDeployment: []*deployments.DeviceDeployment{
				deployments.NewDeviceDeployment("456", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
			},
			OutputError: nil,
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
			store := NewDeviceDeploymentsStorage(session)

			// deployments are created with status DeviceDeploymentStatusPending
			err := store.InsertMany(testCase.InputDeviceDeployment...)
			assert.NoError(t, err)

			err = store.UpdateDeviceDeploymentLogAvailability(
				testCase.InputDeviceID, testCase.InputDeploymentID, testCase.InputLog)

			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}

			if testCase.InputDeviceDeployment != nil {
				var deployment *deployments.DeviceDeployment
				query := bson.M{
					StorageKeyDeviceDeploymentDeviceId:     testCase.InputDeviceID,
					StorageKeyDeviceDeploymentDeploymentID: testCase.InputDeploymentID,
				}
				err := session.DB(DatabaseName).C(CollectionDevices).
					Find(query).One(&deployment)

				assert.NoError(t, err)
				assert.Equal(t, testCase.InputLog, deployment.IsLogAvailable)
			}

			// Need to close all sessions to be able to call wipe at next test case
			session.Close()
		})
	}
}

func newDeviceDeploymentWithStatus(deviceID string, deploymentID string, status string) *deployments.DeviceDeployment {
	d := deployments.NewDeviceDeployment(deviceID, deploymentID)
	d.Status = &status
	return d
}

func TestAggregateDeviceDeploymentByStatus(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestAggregateDeviceDeploymentByStatus in short mode.")
	}

	testCases := []struct {
		InputDeploymentID     string
		InputDeviceDeployment []*deployments.DeviceDeployment
		OutputError           error
		OutputStats           deployments.Stats
	}{
		{
			InputDeploymentID:     "ee13ea8b-a6d3-4d4c-99a6-bcfcaebc7ec3",
			InputDeviceDeployment: nil,
			OutputError:           nil,
			OutputStats:           nil,
		},
		{
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputDeviceDeployment: []*deployments.DeviceDeployment{
				newDeviceDeploymentWithStatus("123", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					deployments.DeviceDeploymentStatusFailure),
				newDeviceDeploymentWithStatus("234", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					deployments.DeviceDeploymentStatusFailure),
				newDeviceDeploymentWithStatus("456", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					deployments.DeviceDeploymentStatusSuccess),

				// these 2 count as in progress
				newDeviceDeploymentWithStatus("567", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					deployments.DeviceDeploymentStatusDownloading),
				newDeviceDeploymentWithStatus("678", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					deployments.DeviceDeploymentStatusRebooting),

				newDeviceDeploymentWithStatus("789", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
					deployments.DeviceDeploymentStatusPending),
			},
			OutputError: nil,
			OutputStats: deployments.Stats{
				deployments.DeviceDeploymentStatusPending:     1,
				deployments.DeviceDeploymentStatusSuccess:     1,
				deployments.DeviceDeploymentStatusFailure:     2,
				deployments.DeviceDeploymentStatusRebooting:   1,
				deployments.DeviceDeploymentStatusDownloading: 1,
				deployments.DeviceDeploymentStatusInstalling:  0,
				deployments.DeviceDeploymentStatusNoArtifact:  0,
				deployments.DeviceDeploymentStatusAlreadyInst: 0,
				deployments.DeviceDeploymentStatusAborted:     0,
			},
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			t.Logf("testing case %s %v %d", testCase.InputDeploymentID, testCase.OutputError,
				len(testCase.InputDeviceDeployment))

			// Make sure we start test with empty database
			db.Wipe()

			session := db.Session()
			store := NewDeviceDeploymentsStorage(session)

			err := store.InsertMany(testCase.InputDeviceDeployment...)
			assert.NoError(t, err)

			stats, err := store.AggregateDeviceDeploymentByStatus(testCase.InputDeploymentID)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
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

	input := []*deployments.DeviceDeployment{
		deployments.NewDeviceDeployment("device0001", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
		deployments.NewDeviceDeployment("device0002", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
		deployments.NewDeviceDeployment("device0003", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
		deployments.NewDeviceDeployment("device0004", "30b3e62c-9ec2-4312-a7fa-cff24cc7397b"),
		deployments.NewDeviceDeployment("device0005", "30b3e62c-9ec2-4312-a7fa-cff24cc7397b"),
	}

	// setup db - once for all cases
	db.Wipe()

	session := db.Session()
	store := NewDeviceDeploymentsStorage(session)

	err := store.InsertMany(input...)
	assert.NoError(t, err)

	testCases :=
		map[string]struct {
			caseId string

			inputDeploymentId string
			outputStatuses    []*deployments.DeviceDeployment
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
				outputStatuses:    []*deployments.DeviceDeployment{},
			},
		}

	for testCaseName, tc := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			statuses, err := store.GetDeviceStatusesForDeployment(tc.inputDeploymentId)
			assert.NoError(t, err)

			assert.Equal(t, len(tc.outputStatuses), len(statuses))
			for i, out := range tc.outputStatuses {
				assert.Equal(t, out.DeviceId, statuses[i].DeviceId)
				assert.Equal(t, out.DeploymentId, statuses[i].DeploymentId)
			}
		})
	}

	session.Close()
}

func TestHasDeploymentForDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping GetDeviceStatusesForDeployment in short mode.")
	}

	input := []*deployments.DeviceDeployment{
		deployments.NewDeviceDeployment("device0001", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
		deployments.NewDeviceDeployment("device0002", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
		deployments.NewDeviceDeployment("device0003", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
	}

	// setup db - once for all cases
	db.Wipe()

	session := db.Session()
	store := NewDeviceDeploymentsStorage(session)

	err := store.InsertMany(input...)
	assert.NoError(t, err)

	testCases := []struct {
		deviceID     string
		deploymentID string

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
	}

	for testCaseNumber, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			t.Logf("testing case: %v %v %v %v", tc.deviceID, tc.deploymentID, tc.has, tc.err)

			has, err := store.HasDeploymentForDevice(tc.deploymentID, tc.deviceID)
			if tc.err != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.err.Error())
				assert.False(t, has)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.has, has)
			}
		})
	}

	session.Close()
}

func TestGetDeviceDeploymentStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping GetDeviceDeploymentStatus in short mode.")
	}

	input := []*deployments.DeviceDeployment{
		deployments.NewDeviceDeployment("device0001", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
		deployments.NewDeviceDeployment("device0002", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
		deployments.NewDeviceDeployment("device0003", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
	}

	// setup db - once for all cases
	db.Wipe()

	session := db.Session()
	store := NewDeviceDeploymentsStorage(session)

	err := store.InsertMany(input...)
	assert.NoError(t, err)

	testCases := map[string]struct {
		deviceID     string
		deploymentID string

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
	}

	for testCaseName, tc := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			t.Logf("testing case: %v %v %v", tc.deviceID, tc.deploymentID, tc.status)

			status, err := store.GetDeviceDeploymentStatus(tc.deploymentID, tc.deviceID)
			assert.NoError(t, err)
			assert.Equal(t, tc.status, status)
		})
	}

	session.Close()
}

func TestAbortDeviceDeployments(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestAbortDeviceDeployments in short mode.")
	}

	testCases := map[string]struct {
		InputDeploymentID     string
		InputDeviceDeployment []*deployments.DeviceDeployment

		OutputError error
	}{
		"null deployment id": {
			OutputError: ErrStorageInvalidID,
		},
		"all correct": {
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputDeviceDeployment: []*deployments.DeviceDeployment{
				deployments.NewDeviceDeployment("456", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
				deployments.NewDeviceDeployment("567", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
			},
			OutputError: nil,
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			// Make sure we start test with empty database
			db.Wipe()

			session := db.Session()
			store := NewDeviceDeploymentsStorage(session)

			err := store.InsertMany(testCase.InputDeviceDeployment...)
			assert.NoError(t, err)

			err = store.AbortDeviceDeployments(testCase.InputDeploymentID)

			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}

			if testCase.InputDeviceDeployment != nil {
				// these checks only make sense if there are any deployments in database
				var deploymentList []deployments.DeviceDeployment
				dep := session.DB(DatabaseName).C(CollectionDevices)
				query := bson.M{
					StorageKeyDeviceDeploymentDeploymentID: testCase.InputDeploymentID,
				}
				err := dep.Find(query).All(&deploymentList)
				assert.NoError(t, err)

				if testCase.OutputError != nil {
					for _, deployment := range deploymentList {
						// status must be unchanged in case of errors
						assert.Equal(t, deployments.DeviceDeploymentStatusPending, *deployment.Status)
					}
				} else {
					for _, deployment := range deploymentList {
						assert.Equal(t, deployments.DeviceDeploymentStatusAborted, *deployment.Status)
					}
				}
			}

			// Need to close all sessions to be able to call wipe at next test case
			session.Close()
		})
	}
}
