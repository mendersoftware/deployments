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
	"context"
	"errors"
	"testing"
	"time"

	"fmt"

	"github.com/mendersoftware/deployments/resources/deployments"
	"github.com/mendersoftware/deployments/resources/deployments/controller"
	. "github.com/mendersoftware/deployments/resources/deployments/model"
	"github.com/mendersoftware/deployments/resources/deployments/model/mocks"
	"github.com/mendersoftware/deployments/resources/images"
	. "github.com/mendersoftware/deployments/utils/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const validUUIDv4 = "d50eda0d-2cea-4de1-8d42-9cd3e7e8670d"

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

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			deploymentStorage := new(mocks.DeploymentsStorage)
			deploymentStorage.On("FindByID", testCase.InputDeploymentID).
				Return(testCase.InoutFindByIDDeployment, testCase.InoutFindByIDError)

			model := NewDeploymentModel(DeploymentsModelConfig{DeploymentsStorage: deploymentStorage})

			deployment, err := model.GetDeployment(testCase.InputDeploymentID)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.OutputDeployment, deployment)
		})
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

			OutputError: errors.New("Checking if image is used by active deployment: Storage error"),
		},
		{
			InputID: "ID:1234",
			InputExistAssignedImageWithIDAndStatusesError: errors.New("Storage error"),
			InputExistAssignedImageWithIDAndStatusesFound: true,

			OutputError: errors.New("Checking if image is used by active deployment: Storage error"),
		},
		{
			InputID: "ID:1234",
			InputExistAssignedImageWithIDAndStatusesFound: true,

			OutputBool: true,
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
			deviceDeploymentStorage.On("ExistAssignedImageWithIDAndStatuses", testCase.InputID, mock.AnythingOfType("[]string")).
				Return(testCase.InputExistAssignedImageWithIDAndStatusesFound,
					testCase.InputExistAssignedImageWithIDAndStatusesError)

			model := NewDeploymentModel(DeploymentsModelConfig{DeviceDeploymentsStorage: deviceDeploymentStorage})

			found, err := model.ImageUsedInActiveDeployment(testCase.InputID)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.OutputBool, found)
		})
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

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
			deviceDeploymentStorage.On("ExistAssignedImageWithIDAndStatuses", testCase.InputID, mock.AnythingOfType("[]string")).
				Return(testCase.InputImageUsedInDeploymentFound,
					testCase.InputImageUsedInDeploymentError)

			model := NewDeploymentModel(DeploymentsModelConfig{DeviceDeploymentsStorage: deviceDeploymentStorage})

			found, err := model.ImageUsedInDeployment(testCase.InputID)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.OutputBool, found)
		})
	}

}

func TestDeploymentModelGetDeploymentForDevice(t *testing.T) {

	t.Parallel()

	image := images.NewSoftwareImage(
		validUUIDv4,
		&images.SoftwareImageMetaConstructor{},
		&images.SoftwareImageMetaArtifactConstructor{
			Name: "foo-artifact",
			DeviceTypesCompatible: []string{
				"hammer",
			},
		})

	testCases := []struct {
		InputID string

		InputOlderstDeviceDeployment      *deployments.DeviceDeployment
		InputOlderstDeviceDeploymentError error

		InputGetRequestLink  *images.Link
		InputGetRequestError error

		InputInstalledDeployment deployments.InstalledDeviceDeployment

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
				Image: images.NewSoftwareImage(
					validUUIDv4,
					&images.SoftwareImageMetaConstructor{},
					&images.SoftwareImageMetaArtifactConstructor{
						Name: "foo-artifact",
					}),
				DeploymentId: StringToPointer("ID:678"),
			},
			InputGetRequestError: errors.New("file storage error"),

			OutputError: errors.New("Generating download link for the device: file storage error"),
		},
		{
			InputID: "ID:123",
			InputOlderstDeviceDeployment: &deployments.DeviceDeployment{
				Image:        image,
				DeploymentId: StringToPointer("ID:678"),
			},
			InputGetRequestLink: &images.Link{},

			OutputDeploymentInstructions: &deployments.DeploymentInstructions{
				ID: "ID:678",
				Artifact: deployments.ArtifactDeploymentInstructions{
					ArtifactName:          image.Name,
					Source:                images.Link{},
					DeviceTypesCompatible: image.DeviceTypesCompatible,
				},
			},
		},
		{
			// currently installed artifact is the same as defined by deployment
			InputID: "ID:123",
			InputOlderstDeviceDeployment: &deployments.DeviceDeployment{
				Id:           StringToPointer("ID:device-deployment-123"),
				Image:        image,
				DeploymentId: StringToPointer("ID:678"),
			},
			InputGetRequestLink: &images.Link{},

			InputInstalledDeployment: deployments.InstalledDeviceDeployment{
				Artifact:   image.Name,
				DeviceType: "hammer",
			},
		},
	}

	for testCaseNumber, testCase := range testCases {

		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			deploymentStorage := new(mocks.DeploymentsStorage)

			deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
			deviceDeploymentStorage.On("FindOldestDeploymentForDeviceIDWithStatuses",
				testCase.InputID, mock.AnythingOfType("[]string")).
				Return(testCase.InputOlderstDeviceDeployment,
					testCase.InputOlderstDeviceDeploymentError)

			// if UpdateDeviceDeploymentStatus is ever called, the status
			// will be already-installed

			deviceDeploymentStorage.On("UpdateDeviceDeploymentStatus",
				mock.AnythingOfType("string"), mock.AnythingOfType("string"),
				deployments.DeviceDeploymentStatusAlreadyInst, mock.AnythingOfType("*time.Time")).
				Return("dontcare", nil)
			deviceDeploymentStorage.On("GetDeviceDeploymentStatus",
				mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return("dontcare", nil)

			imageLinker := new(mocks.GetRequester)
			if testCase.InputOlderstDeviceDeployment != nil {

				// Notice: force GetRequest to expect image id returned
				// by FindOldestDeploymentForDeviceIDWithStatuses Just
				// as implementation does, if this changes test will
				// break by panic ;)
				imageLinker.On("GetRequest", testCase.InputOlderstDeviceDeployment.Image.Id,
					DefaultUpdateDownloadLinkExpire, mock.AnythingOfType("string")).
					Return(testCase.InputGetRequestLink, testCase.InputGetRequestError)

				// if deployment is found to be already installed (i.e.
				// case when current installation artifact is the same
				// as device deployment one), deployment will have its
				// statistics updated
				deploymentStorage.On("UpdateStats",
					*testCase.InputOlderstDeviceDeployment.DeploymentId,
					mock.AnythingOfType("string"),
					deployments.DeviceDeploymentStatusAlreadyInst).
					Return(nil)

				deploymentStorage.On("FindByID",
					*testCase.InputOlderstDeviceDeployment.DeploymentId).
					Return(&deployments.Deployment{
						Id:    testCase.InputOlderstDeviceDeployment.DeploymentId,
						Stats: deployments.NewDeviceDeploymentStats(),
					}, nil)

				// if deployment is found to be finished, we need to
				// mock another call
				deploymentStorage.On("Finish",
					*testCase.InputOlderstDeviceDeployment.DeploymentId,
					mock.AnythingOfType("time.Time")).
					Return(nil)
			}

			model := NewDeploymentModel(DeploymentsModelConfig{
				DeviceDeploymentsStorage: deviceDeploymentStorage,
				DeploymentsStorage:       deploymentStorage,
				ImageLinker:              imageLinker,
			})

			out, err := model.GetDeploymentForDeviceWithCurrent(testCase.InputID,
				testCase.InputInstalledDeployment)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
				assert.Nil(t, out)
			} else {
				assert.NoError(t, err)
				if testCase.OutputDeploymentInstructions != nil {
					assert.NotNil(t, out)
					assert.EqualValues(t, testCase.OutputDeploymentInstructions, out)
				} else {
					assert.Nil(t, out)
				}
			}
		})
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

			OutputError: errors.New("Preparing deployment for device: generation error"),
		},
		{
			InputConstructor: &deployments.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputGenerateDeviceDeployment:     &deployments.DeviceDeployment{},
			InputDeploymentStorageInsertError: errors.New("insert error"),

			OutputError: errors.New("Storing deployment data: insert error"),
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
		{
			InputConstructor: &deployments.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputGenerateDeviceDeployment: &deployments.DeviceDeployment{
				Status: StringToPointer(deployments.DeviceDeploymentStatusNoArtifact),
			},

			OutputBody: true,
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			generator := new(mocks.Generator)
			generator.On("Generate", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("*deployments.Deployment")).
				Return(testCase.InputGenerateDeviceDeployment, testCase.InputGenerateError)

			deploymentStorage := new(mocks.DeploymentsStorage)
			deploymentStorage.On("Insert", mock.AnythingOfType("*deployments.Deployment")).
				Return(testCase.InputDeploymentStorageInsertError)
			deploymentStorage.On("Delete", mock.AnythingOfType("string")).
				Return(testCase.InputDeploymentStorageDeleteError)

			deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
			deviceDeploymentStorage.On("InsertMany", mock.AnythingOfType("[]*deployments.DeviceDeployment")).
				Return(testCase.InputDeviceDeploymentStorageInsertManyError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DeploymentsStorage:        deploymentStorage,
				DeviceDeploymentGenerator: generator,
				DeviceDeploymentsStorage:  deviceDeploymentStorage,
			})

			out, err := model.CreateDeployment(context.Background(), testCase.InputConstructor)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
			if testCase.OutputBody {
				assert.NotNil(t, out)
			}
		})
	}

}

func TestDeploymentModelUpdateDeviceDeploymentStatus(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputDeployment *deployments.Deployment
		InputDeviceID   string
		OldStatus       string
		InputStatus     string

		InputDevsStorageError error
		InputDepsStorageError error
		InputDepsFinishError  error
		InputDepsFindError    error

		isFinished bool

		OutputError error
	}{
		{
			InputDeployment: &deployments.Deployment{
				Id: StringToPointer("123"),
				Stats: deployments.Stats{
					deployments.DeviceDeploymentStatusPending: 1,
				},
			},
			InputDeviceID: "123",
			InputStatus:   "installing",
			OldStatus:     "pending",

			InputDevsStorageError: errors.New("device deployments storage issue"),
			InputDepsStorageError: nil,

			OutputError: errors.New("device deployments storage issue"),
		},
		{
			InputDeployment: &deployments.Deployment{
				Id: StringToPointer("234"),
				Stats: deployments.Stats{
					deployments.DeviceDeploymentStatusPending: 1,
				},
			},
			InputDeviceID: "234",
			InputStatus:   "none",
			OldStatus:     "pending",

			InputDevsStorageError: nil,
			InputDepsStorageError: errors.New("deployments storage issue"),

			OutputError: errors.New("deployments storage issue"),
		},
		{
			isFinished: true,
			InputDeployment: &deployments.Deployment{
				Id: StringToPointer("345"),
				Stats: deployments.Stats{
					deployments.DeviceDeploymentStatusSuccess: 1,
				},
			},
			InputDeviceID: "345",
			InputStatus:   "success",
			OldStatus:     "installing",

			InputDevsStorageError: nil,
		},
		{
			isFinished: false,
			InputDeployment: &deployments.Deployment{
				Id: StringToPointer("345"),
				Stats: deployments.Stats{
					deployments.DeviceDeploymentStatusSuccess: 1,
				},
			},
			InputDeviceID: "345",
			InputStatus:   "installing",
			OldStatus:     "installing",

			InputDevsStorageError: nil,
		},
		{
			isFinished: true,
			InputDeployment: &deployments.Deployment{
				Id: StringToPointer("456"),
				Stats: deployments.Stats{
					deployments.DeviceDeploymentStatusAlreadyInst: 1,
				},
			},
			InputDeviceID: "456",
			InputStatus:   "already-installed",
			OldStatus:     "pending",

			InputDevsStorageError: nil,
		},
		{
			isFinished: true,
			InputDeployment: &deployments.Deployment{
				Id: StringToPointer("567"),
				Stats: deployments.Stats{
					deployments.DeviceDeploymentStatusAborted: 1,
				},
			},
			InputDeviceID: "567",
			InputStatus:   "aborted",
			OldStatus:     "pending",
		},
		{
			isFinished: true,
			InputDeployment: &deployments.Deployment{
				Id: StringToPointer("678"),
				Stats: deployments.Stats{
					deployments.DeviceDeploymentStatusSuccess: 1,
				},
			},
			InputDeviceID: "678",
			InputStatus:   "success",
			OldStatus:     "rebooting",

			InputDepsFinishError: errors.New("deployments storage finish issue"),
			OutputError:          errors.New("failed to mark deployment as finished: deployments storage finish issue"),
		},
		{
			isFinished: true,
			InputDeployment: &deployments.Deployment{
				Id: StringToPointer("789"),
				Stats: deployments.Stats{
					deployments.DeviceDeploymentStatusAborted: 1,
				},
			},
			InputDeviceID: "789",
			InputStatus:   "rebooting",
			OldStatus:     "aborted",

			OutputError: controller.ErrDeploymentAborted,
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			t.Logf("testing %s %s %s %v %v", *testCase.InputDeployment.Id, testCase.InputDeviceID,
				testCase.InputStatus, testCase.InputDevsStorageError, testCase.isFinished)

			deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
			deviceDeploymentStorage.On("UpdateDeviceDeploymentStatus",
				testCase.InputDeviceID, *testCase.InputDeployment.Id,
				testCase.InputStatus, mock.AnythingOfType("*time.Time")).
				Return("dontcare", testCase.InputDevsStorageError)
			deviceDeploymentStorage.On("GetDeviceDeploymentStatus",
				*testCase.InputDeployment.Id, testCase.InputDeviceID).
				Return(testCase.OldStatus, testCase.InputDevsStorageError)

			deploymentStorage := new(mocks.DeploymentsStorage)
			deploymentStorage.On("UpdateStats",
				*testCase.InputDeployment.Id, mock.AnythingOfType("string"),
				mock.AnythingOfType("string")).
				Return(testCase.InputDepsStorageError)
			// deployment will be marked as finished when possible, for this we need to
			// mock a couple of additional calls
			deploymentStorage.On("FindByID",
				*testCase.InputDeployment.Id).
				Return(testCase.InputDeployment, testCase.InputDepsFindError)
			deploymentStorage.On("Finish",
				*testCase.InputDeployment.Id, mock.AnythingOfType("time.Time")).
				Return(testCase.InputDepsFinishError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DeploymentsStorage:       deploymentStorage,
				DeviceDeploymentsStorage: deviceDeploymentStorage,
			})

			err := model.UpdateDeviceDeploymentStatus(*testCase.InputDeployment.Id,
				testCase.InputDeviceID, testCase.InputStatus)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)

				if deployments.IsDeviceDeploymentStatusFinished(testCase.InputStatus) {
					// verify that device deployment finish time was passed, finish time is
					// passed as 4th argument to UpdateDeviceDeploymentStatus
					ft, ok := deviceDeploymentStorage.Calls[1].Arguments.Get(3).(*time.Time)
					assert.True(t, ok)
					assert.WithinDuration(t, time.Now(), *ft, time.Second)
				}

				// check that Finish was called
				if testCase.isFinished {
					deploymentStorage.AssertCalled(t, "Finish",
						*testCase.InputDeployment.Id, mock.AnythingOfType("time.Time"))
				} else {

					deploymentStorage.AssertNotCalled(t, "Finish",
						*testCase.InputDeployment.Id, mock.AnythingOfType("time.Time"))
				}
			}
		})
	}

}

func TestGetDeploymentStats(t *testing.T) {

	t.Parallel()

	testCases := []struct {
		InputDeploymentID         string
		InputModelDeploymentStats deployments.Stats
		InputModelError           error

		InoutFindByIDDeployment *deployments.Deployment
		InoutFindByIDError      error

		OutputStats deployments.Stats
		OutputError error
	}{
		{
			InputDeploymentID:         "ID:123",
			InputModelDeploymentStats: nil,

			InoutFindByIDDeployment: new(deployments.Deployment),

			OutputStats: nil,
		},
		{
			InputDeploymentID: "ID:234",
			InputModelError:   errors.New("storage issue"),

			InoutFindByIDDeployment: new(deployments.Deployment),

			OutputError: errors.New("storage issue"),
		},
		{
			InputDeploymentID: "ID:234",
			InputModelError:   errors.New("storage issue"),

			InoutFindByIDDeployment: nil,
		},
		{
			InputDeploymentID: "ID:234",
			InputModelError:   errors.New("storage issue"),

			InoutFindByIDDeployment: new(deployments.Deployment),
			InoutFindByIDError:      errors.New("an error"),

			OutputError: errors.New("checking deployment id: an error"),
		},
		{
			InputDeploymentID:       "ID:345",
			InoutFindByIDDeployment: new(deployments.Deployment),
			InputModelDeploymentStats: deployments.Stats{
				deployments.DeviceDeploymentStatusPending:     2,
				deployments.DeviceDeploymentStatusSuccess:     4,
				deployments.DeviceDeploymentStatusFailure:     1,
				deployments.DeviceDeploymentStatusInstalling:  3,
				deployments.DeviceDeploymentStatusRebooting:   3,
				deployments.DeviceDeploymentStatusDownloading: 3,
				deployments.DeviceDeploymentStatusAlreadyInst: 0,
			},

			OutputStats: deployments.Stats{
				deployments.DeviceDeploymentStatusDownloading: 3,
				deployments.DeviceDeploymentStatusRebooting:   3,
				deployments.DeviceDeploymentStatusInstalling:  3,
				deployments.DeviceDeploymentStatusSuccess:     4,
				deployments.DeviceDeploymentStatusFailure:     1,
				deployments.DeviceDeploymentStatusPending:     2,
				deployments.DeviceDeploymentStatusAlreadyInst: 0,
			},
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {
			t.Logf("testing %s %v %v", testCase.InputDeploymentID,
				testCase.InputModelDeploymentStats, testCase.InputModelError)

			deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
			deviceDeploymentStorage.On("AggregateDeviceDeploymentByStatus",
				testCase.InputDeploymentID).
				Return(testCase.InputModelDeploymentStats, testCase.InputModelError)

			deploymentStorage := new(mocks.DeploymentsStorage)
			deploymentStorage.On("FindByID", testCase.InputDeploymentID).
				Return(testCase.InoutFindByIDDeployment, testCase.InoutFindByIDError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DeploymentsStorage:       deploymentStorage,
				DeviceDeploymentsStorage: deviceDeploymentStorage,
			})

			stats, err := model.GetDeploymentStats(testCase.InputDeploymentID)

			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)

				assert.Equal(t, testCase.OutputStats, stats)
			}
		})
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

	for testCaseName, tc := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			devsDb := new(mocks.DeviceDeploymentStorage)

			devsDb.On("GetDeviceStatusesForDeployment", tc.inDeploymentId).
				Return(tc.devsStorageStatuses, tc.devsStorageErr)

			depsDb := new(mocks.DeploymentsStorage)

			depsDb.On("FindByID", tc.inDeploymentId).
				Return(tc.depsStorageDeployment, tc.depsStorageErr)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DeploymentsStorage:       depsDb,
				DeviceDeploymentsStorage: devsDb,
			})
			statuses, err := model.GetDeviceStatusesForDeployment(tc.inDeploymentId)

			if tc.modelErr != nil {
				assert.EqualError(t, err, tc.modelErr.Error())
			} else {
				assert.NoError(t, err)

				for i, expected := range tc.devsStorageStatuses {
					assert.Equal(t, expected, statuses[i])
				}
			}
		})
	}
}

func TestDeploymentModelSaveDeviceDeploymentLog(t *testing.T) {

	t.Parallel()

	tref := time.Now()
	messages := []deployments.LogMessage{
		{
			Timestamp: &tref,
			Message:   "foo",
			Level:     "notice",
		},
	}
	testCases := []struct {
		InputDeploymentID string
		InputDeviceID     string
		InputLog          []deployments.LogMessage

		InputModelError     error
		InputHasDeployment  bool
		InputHasModelError  error
		InputUpdateLogError error

		OutputError error
	}{
		{
			InputDeploymentID:  "f826484e-1157-4109-af21-304e6d711560",
			InputDeviceID:      "123",
			InputLog:           messages,
			InputModelError:    errors.New("Storage issue"),
			InputHasDeployment: true,

			OutputError: errors.New("Storage issue"),
		},
		{
			InputDeploymentID:  "ID:234",
			InputDeviceID:      "234",
			InputLog:           []deployments.LogMessage{},
			InputModelError:    nil,
			InputHasDeployment: true,

			OutputError: errors.New("Invalid deployment log: DeploymentID: ID:234 does not validate as uuidv4;Messages: non zero value required;"),
		},
		{
			InputDeploymentID:  "f826484e-1157-4109-af21-304e6d711561",
			InputDeviceID:      "345",
			InputLog:           messages,
			InputModelError:    nil,
			InputHasDeployment: false,

			OutputError: errors.New("Deployment not found"),
		},
		{
			InputDeploymentID:   "f826484e-1157-4109-af21-304e6d711562",
			InputDeviceID:       "456",
			InputLog:            messages,
			InputModelError:     nil,
			InputHasDeployment:  true,
			InputUpdateLogError: errors.New("Could not set log availability"),

			OutputError: errors.New("Could not set log availability"),
		},
		{
			InputDeploymentID:  "f826484e-1157-4109-af21-304e6d711563",
			InputDeviceID:      "567",
			InputLog:           messages,
			InputHasDeployment: true,
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			t.Logf("testing %s %s %s %v", testCase.InputDeploymentID, testCase.InputDeviceID,
				testCase.InputLog, testCase.InputModelError)

			deviceDeploymentLogStorage := new(mocks.DeviceDeploymentLogStorage)
			deviceDeploymentLogStorage.On("SaveDeviceDeploymentLog",
				deployments.DeploymentLog{
					DeviceID:     testCase.InputDeviceID,
					DeploymentID: testCase.InputDeploymentID,
					Messages:     testCase.InputLog,
				}).
				Return(testCase.InputModelError)

			deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
			deviceDeploymentStorage.On("HasDeploymentForDevice",
				testCase.InputDeploymentID, testCase.InputDeviceID).
				Return(testCase.InputHasDeployment, testCase.InputHasModelError)
			deviceDeploymentStorage.On("UpdateDeviceDeploymentLogAvailability",
				testCase.InputDeviceID, testCase.InputDeploymentID, true).
				Return(testCase.InputUpdateLogError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DeviceDeploymentsStorage:    deviceDeploymentStorage,
				DeviceDeploymentLogsStorage: deviceDeploymentLogStorage,
			})

			err := model.SaveDeviceDeploymentLog(testCase.InputDeviceID,
				testCase.InputDeploymentID, testCase.InputLog)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeploymentModelLookupDeployment(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		MockDeployments []*deployments.Deployment
		MockError       error

		OutputError       error
		OutputDeployments []*deployments.Deployment
	}{
		"nothing found": {
			MockDeployments:   nil,
			OutputDeployments: []*deployments.Deployment{},
		},
		"error": {
			MockError:   errors.New("bad bad bad"),
			OutputError: errors.New("searching for deployments: bad bad bad"),
		},
		"found deployments": {
			MockDeployments:   []*deployments.Deployment{&deployments.Deployment{Id: StringToPointer("lala")}},
			OutputDeployments: []*deployments.Deployment{&deployments.Deployment{Id: StringToPointer("lala")}},
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			deploymentStorage := new(mocks.DeploymentsStorage)
			deploymentStorage.On("Find", mock.AnythingOfType("deployments.Query")).
				Return(testCase.MockDeployments, testCase.MockError)

			model := NewDeploymentModel(DeploymentsModelConfig{DeploymentsStorage: deploymentStorage})

			deployments, err := model.LookupDeployment(deployments.Query{})
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.OutputDeployments, deployments)
		})
	}
}

func TestDeploymentModelIsDeploymentFinished(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		InputDeploymentID string
		MockDeployment    *deployments.Deployment
		MockError         error

		OutputValue bool
		OutputError error
	}{
		"nothing found": {
			InputDeploymentID: "f826484e-1157-4109-af21-304e6d711561",
			MockDeployment:    nil,
			OutputError:       nil,
			OutputValue:       false,
		},
		"error": {
			InputDeploymentID: "f826484e-1157-4109-af21-304e6d711561",
			MockDeployment:    nil,
			MockError:         errors.New("bad bad bad"),
			OutputError:       errors.New("Searching for unfinished deployment by ID: bad bad bad"),
		},
		"found unfinished deployment": {
			InputDeploymentID: "f826484e-1157-4109-af21-304e6d711561",
			MockDeployment:    &deployments.Deployment{Id: StringToPointer("f826484e-1157-4109-af21-304e6d711561")},
			OutputValue:       true,
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			deploymentStorage := new(mocks.DeploymentsStorage)
			deploymentStorage.On("FindUnfinishedByID", mock.AnythingOfType("string")).
				Return(testCase.MockDeployment, testCase.MockError)

			model := NewDeploymentModel(DeploymentsModelConfig{DeploymentsStorage: deploymentStorage})

			isFinished, err := model.IsDeploymentFinished(testCase.InputDeploymentID)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.OutputValue, isFinished)
		})
	}
}

func TestDeploymentModelAbortDeployment(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		InputDeploymentID string

		AbortDeviceDeploymentsError            error
		AggregateDeviceDeploymentByStatusStats deployments.Stats
		AggregateDeviceDeploymentByStatusError error
		UpdateStatsAndFinishDeploymentError    error

		OutputError error
	}{
		"AbortDeviceDeployments error": {
			InputDeploymentID:           "f826484e-1157-4109-af21-304e6d711561",
			AbortDeviceDeploymentsError: errors.New("AbortDeviceDeploymentsError"),
			OutputError:                 errors.New("AbortDeviceDeploymentsError"),
		},
		"AggregateDeviceDeploymentByStatus error": {
			InputDeploymentID:                      "f826484e-1157-4109-af21-304e6d711561",
			AggregateDeviceDeploymentByStatusError: errors.New("AggregateDeviceDeploymentByStatusError"),
			AggregateDeviceDeploymentByStatusStats: deployments.Stats{},
			OutputError:                            errors.New("AggregateDeviceDeploymentByStatusError"),
		},
		"UpdateStatsAndFinishDeployment error": {
			InputDeploymentID:                      "f826484e-1157-4109-af21-304e6d711561",
			AggregateDeviceDeploymentByStatusStats: deployments.Stats{"aaa": 1},
			UpdateStatsAndFinishDeploymentError:    errors.New("UpdateStatsAndFinishDeploymentError"),
			OutputError:                            errors.New("UpdateStatsAndFinishDeploymentError"),
		},
		"all correct": {
			InputDeploymentID:                      "f826484e-1157-4109-af21-304e6d711561",
			AggregateDeviceDeploymentByStatusStats: deployments.Stats{"aaa": 1},
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
			deploymentStorage := new(mocks.DeploymentsStorage)
			deviceDeploymentStorage.On("AbortDeviceDeployments", mock.AnythingOfType("string")).
				Return(testCase.AbortDeviceDeploymentsError)
			deviceDeploymentStorage.On("AggregateDeviceDeploymentByStatus", mock.AnythingOfType("string")).
				Return(testCase.AggregateDeviceDeploymentByStatusStats, testCase.AggregateDeviceDeploymentByStatusError)
			deploymentStorage.On("UpdateStatsAndFinishDeployment", mock.AnythingOfType("string"), mock.AnythingOfType("deployments.Stats")).
				Return(testCase.UpdateStatsAndFinishDeploymentError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DeploymentsStorage:       deploymentStorage,
				DeviceDeploymentsStorage: deviceDeploymentStorage,
			})

			err := model.AbortDeployment(testCase.InputDeploymentID)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeploymentModelDecommissionDevice(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		InputDeviceId string

		DecommissionDeviceDeploymentsError                   error
		FindAllDeploymentsForDeviceIDWithStatusesDeployments []deployments.DeviceDeployment
		FindAllDeploymentsForDeviceIDWithStatusesError       error
		AggregateDeviceDeploymentByStatusStats               deployments.Stats
		AggregateDeviceDeploymentByStatusError               error
		UpdateStatsAndFinishDeploymentError                  error

		OutputError error
	}{
		"DecommissionDeviceDeployments error": {
			InputDeviceId:                      "foo",
			DecommissionDeviceDeploymentsError: errors.New("DecommissionDeviceDeploymentsError"),
			OutputError:                        errors.New("DecommissionDeviceDeploymentsError"),
		},
		"AggregateDeviceDeploymentByStatus error": {
			InputDeviceId: "foo",
			FindAllDeploymentsForDeviceIDWithStatusesDeployments: []deployments.DeviceDeployment{*deployments.NewDeviceDeployment("foo", "bar")},
			AggregateDeviceDeploymentByStatusError:               errors.New("AggregateDeviceDeploymentByStatusError"),
			AggregateDeviceDeploymentByStatusStats:               deployments.Stats{},
			OutputError:                                          errors.New("AggregateDeviceDeploymentByStatusError"),
		},
		"UpdateStatsAndFinishDeployment error": {
			InputDeviceId: "foo",
			FindAllDeploymentsForDeviceIDWithStatusesDeployments: []deployments.DeviceDeployment{*deployments.NewDeviceDeployment("foo", "bar")},
			AggregateDeviceDeploymentByStatusStats:               deployments.Stats{"aaa": 1},
			UpdateStatsAndFinishDeploymentError:                  errors.New("UpdateStatsAndFinishDeploymentError"),
			OutputError:                                          errors.New("UpdateStatsAndFinishDeploymentError"),
		},
		"all correct": {
			InputDeviceId: "foo",
			FindAllDeploymentsForDeviceIDWithStatusesDeployments: []deployments.DeviceDeployment{*deployments.NewDeviceDeployment("foo", "bar")},
			AggregateDeviceDeploymentByStatusStats:               deployments.Stats{"aaa": 1},
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			deviceDeploymentStorage := new(mocks.DeviceDeploymentStorage)
			deploymentStorage := new(mocks.DeploymentsStorage)
			deviceDeploymentStorage.On("DecommissionDeviceDeployments", mock.AnythingOfType("string")).
				Return(testCase.DecommissionDeviceDeploymentsError)
			deviceDeploymentStorage.On("FindAllDeploymentsForDeviceIDWithStatuses",
				mock.AnythingOfType("string"),
				mock.AnythingOfType("[]string")).
				Return(testCase.FindAllDeploymentsForDeviceIDWithStatusesDeployments,
					testCase.FindAllDeploymentsForDeviceIDWithStatusesError)
			deviceDeploymentStorage.On("AggregateDeviceDeploymentByStatus", mock.AnythingOfType("string")).
				Return(testCase.AggregateDeviceDeploymentByStatusStats, testCase.AggregateDeviceDeploymentByStatusError)
			deploymentStorage.On("UpdateStatsAndFinishDeployment", mock.AnythingOfType("string"), mock.AnythingOfType("deployments.Stats")).
				Return(testCase.UpdateStatsAndFinishDeploymentError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DeploymentsStorage:       deploymentStorage,
				DeviceDeploymentsStorage: deviceDeploymentStorage,
			})

			err := model.DecommissionDevice(testCase.InputDeviceId)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
