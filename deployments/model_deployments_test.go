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

package deployments

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeploymentModelGetDeployment(t *testing.T) {

	testCases := []struct {
		InputDeploymentID       string
		InoutFindByIDDeployment *Deployment
		InoutFindByIDError      error

		OutputError      error
		OutputDeployment *Deployment
	}{
		{
			InputDeploymentID: "123",
		},
		{
			InputDeploymentID:  "123",
			InoutFindByIDError: errors.New("storage error"),

			OutputError: errors.New("Searching for deployment by ID: storage error"),
		},
		{
			InputDeploymentID:       "123",
			InoutFindByIDDeployment: new(Deployment),

			OutputDeployment: new(Deployment),
		},
	}

	for _, testCase := range testCases {

		deploymentStorage := new(MockDeploymentsStorager)
		deploymentStorage.On("FindByID", testCase.InputDeploymentID).
			Return(testCase.InoutFindByIDDeployment, testCase.InoutFindByIDError)

		model := NewDeploymentModel(deploymentStorage, nil, nil, nil)

		deployment, err := model.GetDeployment(testCase.InputDeploymentID)
		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, testCase.OutputDeployment, deployment)
	}
}

func TestDeploymentModelImageUsedInActiveDeployment(t *testing.T) {

	testCases := []struct {
		InputID string

		InputExistAssignedImageWithIDAndStatusesFound bool
		InputExistAssignedImageWithIDAndStatusesError error

		OutputError error
		OutputBool  bool
	}{
		{
			InputID: "ID:1234",
			InputExistAssignedImageWithIDAndStatusesError: errors.New("Storage error"),

			OutputError: errors.New("Checking if image is used by active deplyoment: Storage error"),
		},
		{
			InputID: "ID:1234",
			InputExistAssignedImageWithIDAndStatusesError: errors.New("Storage error"),
			InputExistAssignedImageWithIDAndStatusesFound: true,

			OutputError: errors.New("Checking if image is used by active deplyoment: Storage error"),
		},
		{
			InputID: "ID:1234",
			InputExistAssignedImageWithIDAndStatusesFound: true,

			OutputBool: true,
		},
	}

	for _, testCase := range testCases {

		deviceDeploymentStorage := new(MockDeviceDeploymentStorager)
		deviceDeploymentStorage.On("ExistAssignedImageWithIDAndStatuses", testCase.InputID, mock.AnythingOfType("[]string")).
			Return(testCase.InputExistAssignedImageWithIDAndStatusesFound,
				testCase.InputExistAssignedImageWithIDAndStatusesError)

		model := NewDeploymentModel(nil, nil, deviceDeploymentStorage, nil)

		found, err := model.ImageUsedInActiveDeployment(testCase.InputID)
		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, testCase.OutputBool, found)
	}

}
