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
	"testing"

	"github.com/mendersoftware/deployments/resources/deployments"
	. "github.com/mendersoftware/deployments/resources/deployments/mongo"
	"github.com/stretchr/testify/assert"
)

func TestSaveDeviceDeploymentLog(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestSaveDeviceDeploymentLog in short mode.")
	}

	testCases := []struct {
		InputDeviceDeploymentLog *deployments.DeploymentLog
		InputDeviceID            string
		InputDeploymentID        string
		OutputError              error
	}{
		{
			// null log
			InputDeviceID:     "123",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			OutputError:       ErrStorageInvalidLog,
		},
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
			InputDeviceDeploymentLog: &deployments.DeploymentLog{
				Messages: []deployments.LogMessage{},
			},
			InputDeviceID:     "456",
			InputDeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			OutputError:       nil,
		},
	}

	for _, testCase := range testCases {

		t.Logf("testing case %s %s %v %v",
			testCase.InputDeviceID, testCase.InputDeploymentID,
			testCase.InputDeviceDeploymentLog, testCase.OutputError)

		// Make sure we start test with empty database
		db.Wipe()

		session := db.Session()
		store := NewDeviceDeploymentLogsStorage(session)

		err := store.SaveDeviceDeploymentLog(testCase.InputDeviceID,
			testCase.InputDeploymentID, testCase.InputDeviceDeploymentLog)

		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)
		}

		// Need to close all sessions to be able to call wipe at next test case
		session.Close()
	}
}
