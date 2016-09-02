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
	"time"

	"github.com/mendersoftware/deployments/resources/deployments"
	. "github.com/mendersoftware/deployments/resources/deployments/mongo"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

func parseTime(t *testing.T, value string) *time.Time {
	tm, err := time.Parse(time.RFC3339, value)
	if assert.NoError(t, err) == false {
		t.Fatalf("failed to parse time %s", value)
	}

	return &tm
}

func TestSaveDeviceDeploymentLog(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestSaveDeviceDeploymentLog in short mode.")
	}

	messages := []deployments.LogMessage{
		{
			Level:     "notice",
			Message:   "foo",
			Timestamp: parseTime(t, "2006-01-02T15:04:05-07:00"),
		},
		{
			Level:     "notice",
			Message:   "bar",
			Timestamp: parseTime(t, "2006-01-02T15:05:05-07:00"),
		},
	}
	testCases := []struct {
		InputDeviceDeploymentLog deployments.DeploymentLog
		OutputError              error
	}{
		{
			// null deployment ID
			InputDeviceDeploymentLog: deployments.DeploymentLog{
				DeviceID: "456",
				Messages: messages,
			},
			OutputError: errors.New("DeploymentID: non zero value required;"),
		},
		{
			// null device ID
			InputDeviceDeploymentLog: deployments.DeploymentLog{
				Messages:     messages,
				DeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			},
			OutputError: errors.New("DeviceID: non zero value required;"),
		},
		{
			InputDeviceDeploymentLog: deployments.DeploymentLog{
				DeviceID:     "456",
				DeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
				Messages:     []deployments.LogMessage{},
			},
			OutputError: nil,
		},
		{
			InputDeviceDeploymentLog: deployments.DeploymentLog{
				DeviceID:     "567",
				DeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",
				Messages:     messages,
			},
			OutputError: nil,
		},
	}

	for _, testCase := range testCases {

		t.Logf("testing case %v %v",
			testCase.InputDeviceDeploymentLog, testCase.OutputError)

		// Make sure we start test with empty database
		db.Wipe()

		session := db.Session()
		store := NewDeviceDeploymentLogsStorage(session)

		err := store.SaveDeviceDeploymentLog(testCase.InputDeviceDeploymentLog)

		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)

			// no errors, so we should be able to find the log in DB
			var dlog deployments.DeploymentLog
			err := session.DB(DatabaseName).C(CollectionDeviceDeploymentLogs).Find(bson.M{
				StorageKeyDeviceDeploymentDeviceId:     testCase.InputDeviceDeploymentLog.DeviceID,
				StorageKeyDeviceDeploymentDeploymentID: testCase.InputDeviceDeploymentLog.DeploymentID,
			}).One(&dlog)

			assert.NoError(t, err)

			// message timestamp is a pointer, so we cannot use assert.EqualValues()
			// or reflect.DeepEqual() as both will choke on *time.Time pointing to
			// different, but value-equal instances, just compare if length is ok for now
			assert.Len(t, dlog.Messages, len(testCase.InputDeviceDeploymentLog.Messages))
		}
		// Need to close all sessions to be able to call wipe at next test case
		session.Close()
	}
}

func TestGetDeviceDeploymentLog(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestGetDeviceDeploymentLog in short mode.")
	}

	messages := []deployments.LogMessage{
		{
			Level:     "notice",
			Message:   "foo",
			Timestamp: parseTime(t, "2006-01-02T15:04:05-07:00"),
		},
		{
			Level:     "notice",
			Message:   "bar",
			Timestamp: parseTime(t, "2006-01-02T15:05:05-07:00"),
		},
	}

	logs := []deployments.DeploymentLog{
		deployments.DeploymentLog{
			DeviceID:     "123",
			DeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			Messages:     messages,
		},
		deployments.DeploymentLog{
			Messages:     messages,
			DeviceID:     "234",
			DeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",
		},
		deployments.DeploymentLog{
			Messages:     messages,
			DeviceID:     "345",
			DeploymentID: "30b3e62c-9ec2-4312-a7fa-cff24cc7397c",
		},
	}

	testCases := []struct {
		InputDeviceID      string
		InputDeploymentID  string
		InputDeploymentLog *deployments.DeploymentLog
		OutputError        error
	}{
		{
			InputDeviceID:      "123",
			InputDeploymentID:  "30b3e62c-9ec2-4312-a7fa-cff24cc7397a",
			InputDeploymentLog: &logs[0],
		},
		{
			InputDeviceID:      "234",
			InputDeploymentID:  "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",
			InputDeploymentLog: &logs[1],
		},
		{
			InputDeviceID:      "345",
			InputDeploymentID:  "30b3e62c-9ec2-4312-a7fa-cff24cc7397c",
			InputDeploymentLog: &logs[2],
		},
		// not found
		{
			InputDeviceID:      "456",
			InputDeploymentID:  "30b3e62c-9ec2-4312-a7fa-cff24cc7397c",
			InputDeploymentLog: nil,
		},
	}

	// Make sure we start test with empty database
	db.Wipe()

	session := db.Session()

	store := NewDeviceDeploymentLogsStorage(session)

	for _, dl := range logs {
		err := store.SaveDeviceDeploymentLog(dl)
		assert.NoError(t, err)
	}

	for _, testCase := range testCases {

		t.Logf("testing case %v %v",
			testCase.InputDeploymentLog, testCase.OutputError)

		dlog, err := store.GetDeviceDeploymentLog(testCase.InputDeviceID, testCase.InputDeploymentID)
		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)

			if testCase.InputDeploymentLog == nil {
				assert.Nil(t, dlog)
			} else {
				assert.Equal(t, testCase.InputDeploymentID, dlog.DeploymentID)
				assert.Equal(t, testCase.InputDeviceID, dlog.DeviceID)
				// message timestamp is a pointer, so we cannot use assert.EqualValues()
				// or reflect.DeepEqual() as both will choke on *time.Time pointing to
				// different, but value-equal instances
				assert.Len(t, dlog.Messages, len(testCase.InputDeploymentLog.Messages))
				for i, m := range testCase.InputDeploymentLog.Messages {
					assert.True(t, m.Timestamp.Equal(*dlog.Messages[i].Timestamp))
					assert.Equal(t, m.Level, dlog.Messages[i].Level)
					assert.Equal(t, m.Message, dlog.Messages[i].Message)
				}
			}
		}
	}
	// Need to close all sessions to be able to call wipe at next test case
	session.Close()

	db.Wipe()
}
