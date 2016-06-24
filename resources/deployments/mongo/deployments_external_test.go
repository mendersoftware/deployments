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
	. "github.com/mendersoftware/deployments/utils/pointers"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentStorageInsert(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestDeploymentStorageInsert in short mode.")
	}

	testCases := []struct {
		InputDeployment *deployments.Deployment
		OutputError     error
	}{
		{
			InputDeployment: nil,
			OutputError:     ErrDeploymentStorageInvalidDeployment,
		},
		{
			InputDeployment: deployments.NewDeployment(),
			OutputError:     errors.New("DeploymentConstructor: non zero value required;"),
		},
		{
			InputDeployment: deployments.NewDeploymentFromConstructor(&deployments.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			}),
			OutputError: nil,
		},
	}

	for _, testCase := range testCases {

		// Make sure we start test with empty database
		db.Wipe()

		session := db.Session()
		store := NewDeploymentsStorage(session)

		err := store.Insert(testCase.InputDeployment)

		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)

			dep := session.DB(DatabaseName).C(CollectionDeployments)
			count, err := dep.Find(nil).Count()
			assert.NoError(t, err)
			assert.Equal(t, 1, count)
		}

		// Need to close all sessions to be able to call wipe at next test case
		session.Close()
	}
}

func TestDeploymentStorageDelete(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestDeploymentStorageDelete in short mode.")
	}

	testCases := []struct {
		InputID                    string
		InputDeplyomentsCollection []interface{}

		OutputError error
	}{
		{
			InputID:     "",
			OutputError: ErrStorageInvalidID,
		},
		{
			InputID:     "b532b01a-9313-404f-8d19-e7fcbe5cc347",
			OutputError: nil,
		},
		{
			InputID: "b532b01a-9313-404f-8d19-e7fcbe5cc347",
			InputDeplyomentsCollection: []interface{}{
				deployments.Deployment{
					DeploymentConstructor: &deployments.DeploymentConstructor{
						Name:         StringToPointer("NYC Production"),
						ArtifactName: StringToPointer("App 123"),
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id: StringToPointer("b532b01a-9313-404f-8d19-e7fcbe5cc347"),
				},
			},
			OutputError: nil,
		},
	}

	for _, testCase := range testCases {

		// Make sure we start test with empty database
		db.Wipe()

		session := db.Session()
		store := NewDeploymentsStorage(session)

		dep := session.DB(DatabaseName).C(CollectionDeployments)
		if testCase.InputDeplyomentsCollection != nil {
			assert.NoError(t, dep.Insert(testCase.InputDeplyomentsCollection...))
		}

		err := store.Delete(testCase.InputID)

		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)

			count, err := dep.FindId(testCase.InputID).Count()
			assert.NoError(t, err)
			assert.Equal(t, 0, count)
		}

		// Need to close all sessions to be able to call wipe at next test case
		session.Close()
	}
}

func TestDeploymentStorageFindByID(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping TestDeploymentStorageFindByID in short mode.")
	}

	testCases := []struct {
		InputID                    string
		InputDeplyomentsCollection []interface{}

		OutputError      error
		OutputDeployment *deployments.Deployment
	}{
		{
			InputID:     "",
			OutputError: ErrStorageInvalidID,
		},
		{
			InputID:          "b532b01a-9313-404f-8d19-e7fcbe5cc347",
			OutputError:      nil,
			OutputDeployment: nil,
		},
		{
			InputID: "b532b01a-9313-404f-8d19-e7fcbe5cc347",
			InputDeplyomentsCollection: []interface{}{
				&deployments.Deployment{
					DeploymentConstructor: &deployments.DeploymentConstructor{
						Name:         StringToPointer("NYC Production"),
						ArtifactName: StringToPointer("App 123"),
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id: StringToPointer("a108ae14-bb4e-455f-9b40-2ef4bab97bb7"),
				},
				&deployments.Deployment{
					DeploymentConstructor: &deployments.DeploymentConstructor{
						Name:         StringToPointer("NYC Production"),
						ArtifactName: StringToPointer("App 123"),
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id: StringToPointer("d1804903-5caa-4a73-a3ae-0efcc3205405"),
				},
			},
			OutputError:      nil,
			OutputDeployment: nil,
		},
		{
			InputID: "a108ae14-bb4e-455f-9b40-2ef4bab97bb7",
			InputDeplyomentsCollection: []interface{}{
				&deployments.Deployment{
					DeploymentConstructor: &deployments.DeploymentConstructor{
						Name:         StringToPointer("NYC Production"),
						ArtifactName: StringToPointer("App 123"),
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id: StringToPointer("a108ae14-bb4e-455f-9b40-2ef4bab97bb7"),
				},
				&deployments.Deployment{
					DeploymentConstructor: &deployments.DeploymentConstructor{
						Name:         StringToPointer("NYC Production"),
						ArtifactName: StringToPointer("App 123"),
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id: StringToPointer("d1804903-5caa-4a73-a3ae-0efcc3205405"),
				},
			},
			OutputError: nil,
			OutputDeployment: &deployments.Deployment{
				DeploymentConstructor: &deployments.DeploymentConstructor{
					Name:         StringToPointer("NYC Production"),
					ArtifactName: StringToPointer("App 123"),
					//Devices is not kept around!
				},
				Id: StringToPointer("a108ae14-bb4e-455f-9b40-2ef4bab97bb7"),
			},
		},
	}

	for _, testCase := range testCases {

		// Make sure we start test with empty database
		db.Wipe()

		session := db.Session()
		store := NewDeploymentsStorage(session)

		dep := session.DB(DatabaseName).C(CollectionDeployments)
		if testCase.InputDeplyomentsCollection != nil {
			assert.NoError(t, dep.Insert(testCase.InputDeplyomentsCollection...))
		}

		deployment, err := store.FindByID(testCase.InputID)

		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)
			assert.Equal(t, testCase.OutputDeployment, deployment)
		}

		// Need to close all sessions to be able to call wipe at next test case
		session.Close()
	}
}
