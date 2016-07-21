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

package model_test

import (
	"errors"
	"testing"

	"github.com/mendersoftware/deployments/resources/deployments"
	"github.com/mendersoftware/deployments/resources/deployments/controller"
	. "github.com/mendersoftware/deployments/resources/deployments/model"
	"github.com/mendersoftware/deployments/resources/deployments/model/mocks"
	"github.com/mendersoftware/deployments/resources/images"
	. "github.com/mendersoftware/deployments/utils/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeploymentModelGetDeployment(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputDeploymentID       string
		InoutFindByIDDeployment *deployments.Deployment
		InoutFindByIDError      error

		OutputError      error
		OutputDeployment *deployments.Deployment
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
			InoutFindByIDDeployment: new(deployments.Deployment),

			OutputDeployment: new(deployments.Deployment),
		},
	}

	for _, testCase := range testCases {

		deploymentStorage := new(mocks.DeploymentsStorage)
		deploymentStorage.On("FindByID", testCase.InputDeploymentID).
			Return(testCase.InoutFindByIDDeployment, testCase.InoutFindByIDError)

		model := NewDeploymentModel(deploymentStorage, nil, nil, nil, nil)

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

	t.Parallel()

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

		deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
		deviceDeploymentStorage.On("ExistAssignedImageWithIDAndStatuses", testCase.InputID, mock.AnythingOfType("[]string")).
			Return(testCase.InputExistAssignedImageWithIDAndStatusesFound,
				testCase.InputExistAssignedImageWithIDAndStatusesError)

		model := NewDeploymentModel(nil, nil, deviceDeploymentStorage, nil, nil)

		found, err := model.ImageUsedInActiveDeployment(testCase.InputID)
		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, testCase.OutputBool, found)
	}

}

func TestDeploymentModelImageUsedInDeployment(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputID string

		InputImageUsedInDeploymentFound bool
		InputImageUsedInDeploymentError error

		OutputError error
		OutputBool  bool
	}{
		{
			InputID: "ID:1234",
			InputImageUsedInDeploymentError: errors.New("Storage error"),

			OutputError: errors.New("Checking if image is used in deployment: Storage error"),
		},
		{
			InputID: "ID:1234",
			InputImageUsedInDeploymentError: errors.New("Storage error"),
			InputImageUsedInDeploymentFound: true,

			OutputError: errors.New("Checking if image is used in deployment: Storage error"),
		},
		{
			InputID: "ID:1234",
			InputImageUsedInDeploymentFound: true,

			OutputBool: true,
		},
	}

	for _, testCase := range testCases {

		deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
		deviceDeploymentStorage.On("ExistAssignedImageWithIDAndStatuses", testCase.InputID, mock.AnythingOfType("[]string")).
			Return(testCase.InputImageUsedInDeploymentFound,
				testCase.InputImageUsedInDeploymentError)

		model := NewDeploymentModel(nil, nil, deviceDeploymentStorage, nil, nil)

		found, err := model.ImageUsedInDeployment(testCase.InputID)
		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, testCase.OutputBool, found)
	}

}

func TestDeploymentModelGetDeploymentForDevice(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputID string

		InputOlderstDeviceDeployment      *deployments.DeviceDeployment
		InputOlderstDeviceDeploymentError error

		InputGetRequestLink  *images.Link
		InputGetRequestError error

		OutputError                  error
		OutputDeploymentInstructions *deployments.DeploymentInstructions
	}{
		{
			InputID: "ID:123",
			InputOlderstDeviceDeploymentError: errors.New("storage issue"),

			OutputError: errors.New("Searching for oldest active deployment for the device: storage issue"),
		},
		{
			InputID: "ID:123",
			// Setting nils just to make it more expressive which case is tested here
			OutputError:                  nil,
			OutputDeploymentInstructions: nil,
		},
		{
			InputID: "ID:123",
			InputOlderstDeviceDeployment: &deployments.DeviceDeployment{
				Image: &images.SoftwareImage{
					Id: StringToPointer("ID:456"),
				},
			},
			InputGetRequestError: errors.New("file storage error"),

			OutputError: errors.New("Generating download link for the device: file storage error"),
		},
		{
			InputID: "ID:123",
			InputOlderstDeviceDeployment: &deployments.DeviceDeployment{
				Image: &images.SoftwareImage{
					Id: StringToPointer("ID:456"),
				},
				Id: StringToPointer("ID:678"),
			},
			InputGetRequestLink: &images.Link{},

			OutputDeploymentInstructions: deployments.NewDeploymentInstructions(
				"ID:678",
				&images.Link{},
				&images.SoftwareImage{
					Id: StringToPointer("ID:456"),
				},
			),
		},
	}

	for _, testCase := range testCases {

		deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
		deviceDeploymentStorage.On("FindOldestDeploymentForDeviceIDWithStatuses", testCase.InputID, mock.AnythingOfType("[]string")).
			Return(testCase.InputOlderstDeviceDeployment,
				testCase.InputOlderstDeviceDeploymentError)

		imageLinker := new(mocks.GetRequester)
		// Notice: force GetRequest to expect image id returned by FindOldestDeploymentForDeviceIDWithStatuses
		//         Just as implementation does, if this changes test will break by panic ;)
		imageLinker.On("GetRequest", "ID:456", DefaultUpdateDownloadLinkExpire).
			Return(testCase.InputGetRequestLink, testCase.InputGetRequestError)

		model := NewDeploymentModel(nil, nil, deviceDeploymentStorage, nil, imageLinker)

		out, err := model.GetDeploymentForDevice(testCase.InputID)
		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, testCase.OutputDeploymentInstructions, out)
	}

}

func TestDeploymentModelCreateDeployment(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputConstructor *deployments.DeploymentConstructor

		InputGenerateDeviceDeployment *deployments.DeviceDeployment
		InputGenerateError            error

		InputDeploymentStorageInsertError           error
		InputDeviceDeploymentStorageInsertManyError error
		InputDeploymentStorageDeleteError           error

		OutputError error
		OutputBody  bool
	}{
		{
			OutputError: controller.ErrModelMissingInput,
		},
		{
			InputConstructor: deployments.NewDeploymentConstructor(),
			OutputError:      errors.New("Validating deployment: Name: non zero value required;ArtifactName: non zero value required;Devices: non zero value required;"),
		},
		{
			InputConstructor: &deployments.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputGenerateError: errors.New("generation error"),

			OutputError: errors.New("Prepring deplyoment for device: generation error"),
		},
		{
			InputConstructor: &deployments.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputGenerateDeviceDeployment:     &deployments.DeviceDeployment{},
			InputDeploymentStorageInsertError: errors.New("insert error"),

			OutputError: errors.New("Storing deplyoment data: insert error"),
		},
		{
			InputConstructor: &deployments.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputGenerateDeviceDeployment:               &deployments.DeviceDeployment{},
			InputDeviceDeploymentStorageInsertManyError: errors.New("insert error"),
			InputDeploymentStorageDeleteError:           errors.New("delete error"),

			OutputError: errors.New("Storing assigned deployments to devices: delete error: insert error"),
		},
		{
			InputConstructor: &deployments.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputGenerateDeviceDeployment:               &deployments.DeviceDeployment{},
			InputDeviceDeploymentStorageInsertManyError: errors.New("insert error"),

			OutputError: errors.New("Storing assigned deployments to devices: insert error"),
		},
		{
			InputConstructor: &deployments.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputGenerateDeviceDeployment: &deployments.DeviceDeployment{},

			OutputBody: true,
		},
	}

	for _, testCase := range testCases {

		generator := new(mocks.Generator)
		generator.On("Generate", mock.AnythingOfType("string"), mock.AnythingOfType("*deployments.Deployment")).
			Return(testCase.InputGenerateDeviceDeployment, testCase.InputGenerateError)

		deploymentStorage := new(mocks.DeploymentsStorage)
		deploymentStorage.On("Insert", mock.AnythingOfType("*deployments.Deployment")).
			Return(testCase.InputDeploymentStorageInsertError)
		deploymentStorage.On("Delete", mock.AnythingOfType("string")).
			Return(testCase.InputDeploymentStorageDeleteError)

		deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
		deviceDeploymentStorage.On("InsertMany", mock.AnythingOfType("[]*deployments.DeviceDeployment")).
			Return(testCase.InputDeviceDeploymentStorageInsertManyError)

		model := NewDeploymentModel(deploymentStorage, generator, deviceDeploymentStorage, nil, nil)

		out, err := model.CreateDeployment(testCase.InputConstructor)
		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)
		}
		if testCase.OutputBody {
			assert.NotNil(t, out)
		}
	}

}

func TestDeploymentModelUpdateDeviceDeploymentStatus(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputDeploymentID string
		InputDeviceID     string
		InputStatus       string

		InputModelError error

		OutputError error
	}{
		{
			InputDeploymentID: "ID:123",
			InputDeviceID:     "123",
			InputStatus:       "installing",
			InputModelError:   errors.New("storage issue"),

			OutputError: errors.New("storage issue"),
		},
		{
			InputDeploymentID: "ID:234",
			InputDeviceID:     "234",
			InputStatus:       "none",
			InputModelError:   nil,

			OutputError: nil,
		},
	}

	for _, testCase := range testCases {

		t.Logf("testing %s %s %s %v", testCase.InputDeploymentID, testCase.InputDeviceID,
			testCase.InputStatus, testCase.InputModelError)

		deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
		deviceDeploymentStorage.On("UpdateDeviceDeploymentStatus",
			testCase.InputDeviceID, testCase.InputDeploymentID,
			testCase.InputStatus).
			Return(testCase.InputModelError)

		model := NewDeploymentModel(nil, nil, deviceDeploymentStorage, nil, nil)

		err := model.UpdateDeviceDeploymentStatus(testCase.InputDeploymentID,
			testCase.InputDeviceID, testCase.InputStatus)
		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)
		}
	}

}

func TestGetDeploymentStats(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputDeploymentID         string
		InputModelDeploymentStats deployments.Stats
		InputModelError           error

		OutputStats deployments.Stats
		OutputError error
	}{
		{
			InputDeploymentID:         "ID:123",
			InputModelDeploymentStats: nil,

			OutputStats: nil,
		},
		{
			InputDeploymentID: "ID:234",
			InputModelError:   errors.New("storage issue"),

			OutputError: errors.New("storage issue"),
		},
		{
			InputDeploymentID: "ID:345",
			InputModelDeploymentStats: deployments.Stats{
				deployments.DeviceDeploymentStatusPending:     2,
				deployments.DeviceDeploymentStatusSuccess:     4,
				deployments.DeviceDeploymentStatusFailure:     1,
				deployments.DeviceDeploymentStatusInstalling:  3,
				deployments.DeviceDeploymentStatusRebooting:   3,
				deployments.DeviceDeploymentStatusDownloading: 3,
			},

			OutputStats: deployments.Stats{
				deployments.DeviceDeploymentStatusDownloading: 3,
				deployments.DeviceDeploymentStatusRebooting:   3,
				deployments.DeviceDeploymentStatusInstalling:  3,
				deployments.DeviceDeploymentStatusSuccess:     4,
				deployments.DeviceDeploymentStatusFailure:     1,
				deployments.DeviceDeploymentStatusPending:     2,
			},
		},
	}

	for _, testCase := range testCases {
		t.Logf("testing %s %v %v", testCase.InputDeploymentID,
			testCase.InputModelDeploymentStats, testCase.InputModelError)

		deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
		deviceDeploymentStorage.On("AggregateDeviceDeploymentByStatus",
			testCase.InputDeploymentID).
			Return(testCase.InputModelDeploymentStats, testCase.InputModelError)

		model := NewDeploymentModel(nil, nil, deviceDeploymentStorage, nil, nil)

		stats, err := model.GetDeploymentStats(testCase.InputDeploymentID)
		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)

			assert.Equal(t, testCase.OutputStats, stats)
		}
	}
}

func TestDeploymentModelGetDeviceStatusesForDeployment(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		inDeploymentId string

		devsStorageStatuses []deployments.DeviceDeployment
		devsStorageErr      error

		depsStorageDeployment *deployments.Deployment
		depsStorageErr        error

		modelErr error
	}{
		"existing deployment with statuses": {
			inDeploymentId: "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",

			devsStorageStatuses: []deployments.DeviceDeployment{
				*deployments.NewDeviceDeployment("dev0001", "30b3e62c-9ec2-4312-a7fa-cff24cc7397b"),
				*deployments.NewDeviceDeployment("dev0002", "30b3e62c-9ec2-4312-a7fa-cff24cc7397b"),
				*deployments.NewDeviceDeployment("dev0003", "30b3e62c-9ec2-4312-a7fa-cff24cc7397b"),
			},
			devsStorageErr: nil,

			depsStorageDeployment: &deployments.Deployment{},
			depsStorageErr:        nil,

			modelErr: nil,
		},
		"deployment doesn't exist": {
			devsStorageStatuses: []deployments.DeviceDeployment{
				*deployments.NewDeviceDeployment("dev0001", "30b3e62c-9ec2-4312-a7fa-cff24cc7397b"),
				*deployments.NewDeviceDeployment("dev0002", "30b3e62c-9ec2-4312-a7fa-cff24cc7397b"),
				*deployments.NewDeviceDeployment("dev0003", "30b3e62c-9ec2-4312-a7fa-cff24cc7397b"),
			},
			devsStorageErr: nil,

			depsStorageDeployment: nil,
			depsStorageErr:        nil,

			modelErr: controller.ErrModelDeploymentNotFound,
		},
		"DeviceDeployments storage layer error": {
			inDeploymentId: "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",

			devsStorageStatuses: nil,
			devsStorageErr:      errors.New("some verbose, low-level db error"),

			depsStorageDeployment: &deployments.Deployment{},
			depsStorageErr:        nil,

			modelErr: errors.New("Internal error"),
		},
		"Deployments storage layer error": {
			inDeploymentId: "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",

			devsStorageStatuses: nil,
			devsStorageErr:      nil,

			depsStorageDeployment: nil,
			depsStorageErr:        errors.New("some verbose, low-level db error"),

			modelErr: errors.New("Internal error"),
		},
	}

	for id, tc := range testCases {
		t.Logf("test case: %s", id)

		devsDb := new(mocks.DeviceDeploymentStorage)

		devsDb.On("GetDeviceStatusesForDeployment", tc.inDeploymentId).
			Return(tc.devsStorageStatuses, tc.devsStorageErr)

		depsDb := new(mocks.DeploymentsStorage)

		depsDb.On("FindByID", tc.inDeploymentId).
			Return(tc.depsStorageDeployment, tc.depsStorageErr)

		model := NewDeploymentModel(depsDb, nil, devsDb, nil, nil)
		statuses, err := model.GetDeviceStatusesForDeployment(tc.inDeploymentId)

		if tc.modelErr != nil {
			assert.EqualError(t, err, tc.modelErr.Error())
		} else {
			assert.NoError(t, err)

			for i, expected := range tc.devsStorageStatuses {
				assert.Equal(t, expected, statuses[i])
			}
		}
	}
}

func TestDeploymentModelSaveDeviceDeploymentLog(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputDeploymentID string
		InputDeviceID     string
		InputLog          *deployments.DeploymentLog

		InputModelError error

		OutputError error
	}{
		{
			InputDeploymentID: "ID:123",
			InputDeviceID:     "123",
			InputLog: &deployments.DeploymentLog{
				Messages: []deployments.LogMessage{},
			},
			InputModelError: errors.New("storage issue"),

			OutputError: errors.New("storage issue"),
		},
		{
			InputDeploymentID: "ID:234",
			InputDeviceID:     "234",
			InputLog: &deployments.DeploymentLog{
				Messages: []deployments.LogMessage{},
			},
			InputModelError: nil,

			OutputError: nil,
		},
	}

	for _, testCase := range testCases {

		t.Logf("testing %s %s %s %v", testCase.InputDeploymentID, testCase.InputDeviceID,
			testCase.InputLog, testCase.InputModelError)

		deviceDeploymentLogStorage := new(mocks.DeviceDeploymentLogStorage)
		deviceDeploymentLogStorage.On("SaveDeviceDeploymentLog",
			testCase.InputDeviceID, testCase.InputDeploymentID,
			testCase.InputLog).
			Return(testCase.InputModelError)

		model := NewDeploymentModel(nil, nil, nil, deviceDeploymentLogStorage, nil)

		err := model.SaveDeviceDeploymentLog(testCase.InputDeviceID,
			testCase.InputDeploymentID, testCase.InputLog)
		if testCase.OutputError != nil {
			assert.EqualError(t, err, testCase.OutputError.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}
