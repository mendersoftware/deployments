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
	"testing"

	"github.com/mendersoftware/deployments/resources/deployments"
	. "github.com/mendersoftware/deployments/resources/deployments/mongo"
	// . "github.com/mendersoftware/deployments/utils/pointers"
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

	for _, testCase := range testCases {

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
	}
}

func TestUpdateDeviceDeploymentStatus(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestUpdateDeviceDeploymentStatus in short mode.")
	}

	testCases := []struct {
		InputDeviceID         string
		InputDeploymentID     string
		InputStatus           string
		InputDeviceDeployment []*deployments.DeviceDeployment
		OutputError           error
	}{
		{
			// null status
			InputDeviceID:     "123",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			OutputError:       ErrStorageInvalidID,
		},
		{
			// null deployment ID
			InputDeviceID: "234",
			InputStatus:   "",
			OutputError:   ErrStorageInvalidID,
		},
		{
			// null device ID
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputStatus:       "notnull",
			OutputError:       ErrStorageInvalidID,
		},
		{
			// no deployment/device with this ID
			InputDeviceID:     "345",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputStatus:       "notnull",
			OutputError:       errors.New("not found"),
		},
		{
			InputDeviceID:     "456",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputStatus:       deployments.DeviceDeploymentStatusInstalling,
			InputDeviceDeployment: []*deployments.DeviceDeployment{
				deployments.NewDeviceDeployment("456", "30b3e62c-9ec2-4312-a7fa-cff24cc7397a"),
			},
			OutputError: nil,
		},
	}

	for _, testCase := range testCases {

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

		err = store.UpdateDeviceDeploymentStatus(testCase.InputDeviceID,
			testCase.InputDeploymentID, testCase.InputStatus)

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
			}
		}

		// Need to close all sessions to be able to call wipe at next test case
		session.Close()
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
		OutputStats           deployments.RawStats
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
			OutputStats: deployments.RawStats{
				deployments.DeviceDeploymentStatusPending:     1,
				deployments.DeviceDeploymentStatusSuccess:     1,
				deployments.DeviceDeploymentStatusFailure:     2,
				deployments.DeviceDeploymentStatusRebooting:   1,
				deployments.DeviceDeploymentStatusDownloading: 1,
			},
		},
	}

	for _, testCase := range testCases {

		t.Logf("testing case %s %v %d", testCase.InputDeploymentID, testCase.OutputError,
			len(testCase.InputDeviceDeployment))

		// Make sure we start test with empty database
		db.Wipe()

		session := db.Session()
		store := NewDeviceDeploymentsStorage(session)

		// deployments are created with status DeviceDeploymentStatusPending
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
	}
}
