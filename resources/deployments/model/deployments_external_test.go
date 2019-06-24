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

package model

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/deployments/resources/deployments/controller"
	dmmocks "github.com/mendersoftware/deployments/resources/deployments/model/mocks"
	"github.com/mendersoftware/deployments/store/mocks"
	. "github.com/mendersoftware/deployments/utils/pointers"
	h "github.com/mendersoftware/deployments/utils/testing"
)

const (
	validUUIDv4  = "d50eda0d-2cea-4de1-8d42-9cd3e7e8670d"
	artifactSize = 10000
)

func TestDeploymentModelGetDeployment(t *testing.T) {

	//t.Parallel()

	testCases := []struct {
		InputDeploymentID       string
		InoutFindByIDDeployment *model.Deployment
		InoutFindByIDError      error

		OutputError      error
		OutputDeployment *model.Deployment
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
			InoutFindByIDDeployment: new(model.Deployment),

			OutputDeployment: new(model.Deployment),
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			db := new(mocks.DataStore)
			db.On("FindDeploymentByID",
				h.ContextMatcher(),
				testCase.InputDeploymentID).
				Return(testCase.InoutFindByIDDeployment, testCase.InoutFindByIDError)

			model := NewDeploymentModel(DeploymentsModelConfig{DataStore: db})

			deployment, err := model.GetDeployment(context.Background(),
				testCase.InputDeploymentID)
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

	//t.Parallel()

	testCases := []struct {
		InputID string

		InputExistAssignedImageWithIDAndStatusesFound bool
		InputExistAssignedImageWithIDAndStatusesError error

		InputExistUnfinishedByArtifactIdFlag bool
		ExistUnfinishedByArtifactIdError     error

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

			db := new(mocks.DataStore)
			db.On("ExistAssignedImageWithIDAndStatuses",
				h.ContextMatcher(),
				testCase.InputID, mock.AnythingOfType("[]string")).
				Return(testCase.InputExistAssignedImageWithIDAndStatusesFound,
					testCase.InputExistAssignedImageWithIDAndStatusesError)

			db.On("ExistUnfinishedByArtifactId",
				h.ContextMatcher(),
				mock.AnythingOfType("string")).
				Return(testCase.InputExistUnfinishedByArtifactIdFlag,
					testCase.ExistUnfinishedByArtifactIdError)

			model := NewDeploymentModel(
				DeploymentsModelConfig{
					DataStore: db,
				})

			found, err := model.ImageUsedInActiveDeployment(context.Background(),
				testCase.InputID)
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

	//t.Parallel()

	testCases := []struct {
		InputID string

		InputImageUsedInDeploymentFound bool
		InputImageUsedInDeploymentError error

		InputExistUnfinishedByArtifactIdFlag bool
		ExistUnfinishedByArtifactIdError     error

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

			db := new(mocks.DataStore)
			db.On("ExistAssignedImageWithIDAndStatuses",
				h.ContextMatcher(),
				testCase.InputID, mock.AnythingOfType("[]string")).
				Return(testCase.InputImageUsedInDeploymentFound,
					testCase.InputImageUsedInDeploymentError)

			db.On("ExistUnfinishedByArtifactId",
				h.ContextMatcher(),
				mock.AnythingOfType("string")).
				Return(testCase.InputExistUnfinishedByArtifactIdFlag,
					testCase.ExistUnfinishedByArtifactIdError)

			model := NewDeploymentModel(
				DeploymentsModelConfig{
					DataStore: db,
				})

			found, err := model.ImageUsedInDeployment(context.Background(),
				testCase.InputID)
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

	//t.Parallel()

	image := model.NewSoftwareImage(
		validUUIDv4,
		&model.SoftwareImageMetaConstructor{},
		&model.SoftwareImageMetaArtifactConstructor{
			Name: "foo-artifact",
			DeviceTypesCompatible: []string{
				"hammer",
			},
		}, artifactSize)

	testCases := []struct {
		InputID string

		InputOlderstDeviceDeployment      *model.DeviceDeployment
		InputOlderstDeviceDeploymentError error

		InputGetRequestLink  *model.Link
		InputGetRequestError error

		InputInstalledDeployment model.InstalledDeviceDeployment

		InputArtifact                      *model.SoftwareImage
		InputImageByIdsAndDeviceTypeError  error
		InputImageByNameAndDeviceTypeError error

		InputAssignArtifactError error

		InputExistUnfinishedByArtifactIdFlag bool
		ExistUnfinishedByArtifactIdError     error

		OutputError                  error
		OutputDeploymentInstructions *model.DeploymentInstructions
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
			InputOlderstDeviceDeployment: &model.DeviceDeployment{
				Image: model.NewSoftwareImage(
					validUUIDv4,
					&model.SoftwareImageMetaConstructor{},
					&model.SoftwareImageMetaArtifactConstructor{
						Name: image.Name,
						DeviceTypesCompatible: []string{
							"hammer",
						},
					}, artifactSize),
				DeviceId:     StringToPointer("ID:123"),
				DeploymentId: StringToPointer("ID:678"),
			},
			InputArtifact: model.NewSoftwareImage(
				validUUIDv4,
				&model.SoftwareImageMetaConstructor{},
				&model.SoftwareImageMetaArtifactConstructor{
					Name: image.Name,
					DeviceTypesCompatible: []string{
						"hammer",
					},
				}, artifactSize),
			InputInstalledDeployment: model.InstalledDeviceDeployment{
				Artifact:   "different-artifact",
				DeviceType: "hammer",
			},
			InputGetRequestError: errors.New("file storage error"),

			OutputError: errors.New("Generating download link for the device: file storage error"),
		},
		{
			InputID: "ID:123",
			InputOlderstDeviceDeployment: &model.DeviceDeployment{
				Image:        image,
				DeviceId:     StringToPointer("ID:123"),
				DeploymentId: StringToPointer("ID:678"),
			},
			InputArtifact:       image,
			InputGetRequestLink: &model.Link{},

			OutputDeploymentInstructions: &model.DeploymentInstructions{
				ID: "ID:678",
				Artifact: model.ArtifactDeploymentInstructions{
					ArtifactName:          image.Name,
					Source:                model.Link{},
					DeviceTypesCompatible: image.DeviceTypesCompatible,
				},
			},
		},
		{
			// currently installed artifact is the same as defined by deployment
			InputID: "ID:123",
			InputOlderstDeviceDeployment: &model.DeviceDeployment{
				Id:           StringToPointer("ID:device-deployment-123"),
				DeviceId:     StringToPointer("ID:123"),
				Image:        image,
				DeploymentId: StringToPointer("ID:678"),
			},
			InputGetRequestLink: &model.Link{},

			InputInstalledDeployment: model.InstalledDeviceDeployment{
				Artifact:   image.Name,
				DeviceType: "hammer",
			},
		},
	}

	for testCaseNumber, testCase := range testCases {

		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			db := new(mocks.DataStore)
			db.On("ExistUnfinishedByArtifactId",
				h.ContextMatcher(),
				mock.AnythingOfType("string")).
				Return(testCase.InputExistUnfinishedByArtifactIdFlag,
					testCase.ExistUnfinishedByArtifactIdError)

			db.On("FindOldestDeploymentForDeviceIDWithStatuses",
				h.ContextMatcher(),
				testCase.InputID, mock.AnythingOfType("[]string")).
				Return(testCase.InputOlderstDeviceDeployment,
					testCase.InputOlderstDeviceDeploymentError)

			db.On("UpdateDeviceDeploymentStatus",
				h.ContextMatcher(),
				mock.AnythingOfType("string"), mock.AnythingOfType("string"),
				mock.AnythingOfType("model.DeviceDeploymentStatus")).
				Return("dontcare", nil)
			db.On("GetDeviceDeploymentStatus",
				h.ContextMatcher(),
				mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return("dontcare", nil)

			db.On("AssignArtifact",
				h.ContextMatcher(),
				mock.AnythingOfType("string"),
				mock.AnythingOfType("string"),
				mock.AnythingOfType("*model.SoftwareImage")).
				Return(nil)
				//Return(testCase.InputAssignArtifactError)

			imageLinker := new(dmmocks.GetRequester)
			if testCase.InputOlderstDeviceDeployment != nil {

				// Notice: force GetRequest to expect image id returned
				// by FindOldestDeploymentForDeviceIDWithStatuses Just
				// as implementation does, if this changes test will
				// break by panic ;)
				imageLinker.On("GetRequest", h.ContextMatcher(),
					testCase.InputOlderstDeviceDeployment.Image.Id,
					DefaultUpdateDownloadLinkExpire, mock.AnythingOfType("string")).
					Return(testCase.InputGetRequestLink, testCase.InputGetRequestError)

				// if deployment is found to be already installed (i.e.
				// case when current installation artifact is the same
				// as device deployment one), deployment will have its
				// statistics updated
				db.On("UpdateStats",
					h.ContextMatcher(),
					*testCase.InputOlderstDeviceDeployment.DeploymentId,
					mock.AnythingOfType("string"),
					mock.AnythingOfType("string")).
					Return(nil)

				db.On("FindDeploymentByID",
					h.ContextMatcher(),
					*testCase.InputOlderstDeviceDeployment.DeploymentId).
					Return(&model.Deployment{
						Id:    testCase.InputOlderstDeviceDeployment.DeploymentId,
						Stats: model.NewDeviceDeploymentStats(),
						DeploymentConstructor: &model.DeploymentConstructor{
							ArtifactName: &image.Name,
						},
					}, nil)

				// if deployment is found to be finished, we need to
				// mock another call
				db.On("Finish",
					h.ContextMatcher(),
					*testCase.InputOlderstDeviceDeployment.DeploymentId,
					mock.AnythingOfType("time.Time")).
					Return(nil)
			}

			db.On("ImageByIdsAndDeviceType",
				h.ContextMatcher(),
				mock.AnythingOfType("[]string"),
				mock.AnythingOfType("string")).
				Return(testCase.InputArtifact,
					testCase.InputImageByIdsAndDeviceTypeError)

			db.On("ImageByNameAndDeviceType",
				h.ContextMatcher(),
				mock.AnythingOfType("string"),
				mock.AnythingOfType("string")).
				Return(testCase.InputArtifact,
					testCase.InputImageByNameAndDeviceTypeError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DataStore:   db,
				ImageLinker: imageLinker,
			})

			out, err := model.GetDeploymentForDeviceWithCurrent(context.Background(),
				testCase.InputID,
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

	//t.Parallel()

	testCases := []struct {
		InputConstructor *model.DeploymentConstructor

		InputDeploymentStorageInsertError           error
		InputDeviceDeploymentStorageInsertManyError error
		InputDeploymentStorageDeleteError           error
		InputImagesByNameError                      error

		OutputError error
		OutputBody  bool
	}{
		{
			OutputError: controller.ErrModelMissingInput,
		},
		{
			InputConstructor: &model.DeploymentConstructor{},
			OutputError:      errors.New("Validating deployment: name: non zero value required;artifact_name: non zero value required;devices: non zero value required"),
		},
		{
			InputConstructor: &model.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputDeploymentStorageInsertError: errors.New("insert error"),

			OutputError: errors.New("Storing deployment data: insert error"),
		},
		{
			InputConstructor: &model.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputDeviceDeploymentStorageInsertManyError: errors.New("insert error"),
			InputDeploymentStorageDeleteError:           errors.New("delete error"),

			OutputError: errors.New("Storing assigned deployments to devices: delete error: insert error"),
		},
		{
			InputConstructor: &model.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputDeviceDeploymentStorageInsertManyError: errors.New("insert error"),

			OutputError: errors.New("Storing assigned deployments to devices: insert error"),
		},
		{
			InputConstructor: &model.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},

			OutputBody: true,
		},
		{
			InputConstructor: &model.DeploymentConstructor{
				Name:         StringToPointer("NYC Production"),
				ArtifactName: StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},

			OutputBody: true,
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {

			db := new(mocks.DataStore)
			db.On("InsertDeployment",
				h.ContextMatcher(),
				mock.AnythingOfType("*model.Deployment")).
				Return(testCase.InputDeploymentStorageInsertError)
			db.On("DeleteDeployment",
				h.ContextMatcher(),
				mock.AnythingOfType("string")).
				Return(testCase.InputDeploymentStorageDeleteError)

			db.On("InsertMany",
				h.ContextMatcher(),
				mock.AnythingOfType("[]*model.DeviceDeployment")).
				Return(testCase.InputDeviceDeploymentStorageInsertManyError)

			db.On("ImagesByName",
				h.ContextMatcher(),
				mock.AnythingOfType("string")).
				Return(
					[]*model.SoftwareImage{model.NewSoftwareImage(
						validUUIDv4,
						&model.SoftwareImageMetaConstructor{},
						&model.SoftwareImageMetaArtifactConstructor{
							//Name: *testCase.InputConstructor.ArtifactName,
							Name: "App 123",
							DeviceTypesCompatible: []string{
								"hammer",
							},
						}, artifactSize)},
					testCase.InputImagesByNameError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DataStore: db,
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

	//t.Parallel()

	testCases := []struct {
		InputDeployment *model.Deployment
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
			InputDeployment: &model.Deployment{
				Id: StringToPointer("123"),
				Stats: model.Stats{
					model.DeviceDeploymentStatusPending: 1,
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
			InputDeployment: &model.Deployment{
				Id: StringToPointer("234"),
				Stats: model.Stats{
					model.DeviceDeploymentStatusPending: 1,
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
			InputDeployment: &model.Deployment{
				Id: StringToPointer("345"),
				Stats: model.Stats{
					model.DeviceDeploymentStatusSuccess: 1,
				},
			},
			InputDeviceID: "345",
			InputStatus:   "success",
			OldStatus:     "installing",

			InputDevsStorageError: nil,
		},
		{
			isFinished: false,
			InputDeployment: &model.Deployment{
				Id: StringToPointer("345"),
				Stats: model.Stats{
					model.DeviceDeploymentStatusSuccess: 1,
				},
			},
			InputDeviceID: "345",
			InputStatus:   "installing",
			OldStatus:     "installing",

			InputDevsStorageError: nil,
		},
		{
			isFinished: true,
			InputDeployment: &model.Deployment{
				Id: StringToPointer("456"),
				Stats: model.Stats{
					model.DeviceDeploymentStatusAlreadyInst: 1,
				},
			},
			InputDeviceID: "456",
			InputStatus:   "already-installed",
			OldStatus:     "pending",

			InputDevsStorageError: nil,
		},
		{
			isFinished: true,
			InputDeployment: &model.Deployment{
				Id: StringToPointer("567"),
				Stats: model.Stats{
					model.DeviceDeploymentStatusAborted: 1,
				},
			},
			InputDeviceID: "567",
			InputStatus:   "aborted",
			OldStatus:     "pending",
		},
		{
			isFinished: true,
			InputDeployment: &model.Deployment{
				Id: StringToPointer("678"),
				Stats: model.Stats{
					model.DeviceDeploymentStatusSuccess: 1,
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
			InputDeployment: &model.Deployment{
				Id: StringToPointer("789"),
				Stats: model.Stats{
					model.DeviceDeploymentStatusAborted: 1,
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

			t.Logf("testing %s %s %s (in) %s (old) %v %v",
				*testCase.InputDeployment.Id,
				testCase.InputDeviceID, testCase.InputStatus,
				testCase.OldStatus,
				testCase.InputDevsStorageError, testCase.isFinished)

			// status is always fetched
			db := new(mocks.DataStore)
			db.On("GetDeviceDeploymentStatus",
				h.ContextMatcher(),
				*testCase.InputDeployment.Id, testCase.InputDeviceID).
				Return(testCase.OldStatus, testCase.InputDevsStorageError)

			// however any status updates in DB are done only if the
			// status reported by device is different from the
			// previous one in DB
			if testCase.OldStatus != testCase.InputStatus {
				db.On("UpdateDeviceDeploymentStatus",
					h.ContextMatcher(),
					testCase.InputDeviceID, *testCase.InputDeployment.Id,
					mock.MatchedBy(func(ddStatus model.DeviceDeploymentStatus) bool {

						statusOk := assert.Equal(t, testCase.InputStatus, ddStatus.Status)
						finishOk := true
						if testCase.isFinished {
							finishOk = assert.NotNil(t, ddStatus.FinishTime) &&
								assert.WithinDuration(t, time.Now(),
									*ddStatus.FinishTime, time.Second)
						}
						return statusOk && finishOk
					})).
					Return("dontcare", testCase.InputDevsStorageError)

				db.On("UpdateStats",
					h.ContextMatcher(),
					*testCase.InputDeployment.Id, mock.AnythingOfType("string"),
					mock.AnythingOfType("string")).
					Return(testCase.InputDepsStorageError)
				// deployment will be marked as finished when possible, for this we need to
				// mock a couple of additional calls
				db.On("FindDeploymentByID",
					h.ContextMatcher(),
					*testCase.InputDeployment.Id).
					Return(testCase.InputDeployment, testCase.InputDepsFindError)
				if testCase.isFinished {
					db.On("Finish",
						h.ContextMatcher(),
						*testCase.InputDeployment.Id,
						mock.MatchedBy(func(tm time.Time) bool {
							return assert.WithinDuration(t, time.Now(), tm, time.Second)
						})).
						Return(testCase.InputDepsFinishError)
				}
			}

			dmodel := NewDeploymentModel(DeploymentsModelConfig{
				DataStore: db,
			})

			err := dmodel.UpdateDeviceDeploymentStatus(context.Background(),
				*testCase.InputDeployment.Id,
				testCase.InputDeviceID,
				model.DeviceDeploymentStatus{
					Status: testCase.InputStatus,
				})
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)

				// make sure that storage calls were done as
				// expected
				mock.AssertExpectationsForObjects(t, db)
			}
		})
	}

}

func TestGetDeploymentStats(t *testing.T) {

	//t.Parallel()

	testCases := []struct {
		InputDeploymentID         string
		InputModelDeploymentStats model.Stats
		InputModelError           error

		InoutFindByIDDeployment *model.Deployment
		InoutFindByIDError      error

		OutputStats model.Stats
		OutputError error
	}{
		{
			InputDeploymentID:         "ID:123",
			InputModelDeploymentStats: nil,

			InoutFindByIDDeployment: new(model.Deployment),

			OutputStats: nil,
		},
		{
			InputDeploymentID: "ID:234",
			InputModelError:   errors.New("storage issue"),

			InoutFindByIDDeployment: new(model.Deployment),

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

			InoutFindByIDDeployment: new(model.Deployment),
			InoutFindByIDError:      errors.New("an error"),

			OutputError: errors.New("checking deployment id: an error"),
		},
		{
			InputDeploymentID:       "ID:345",
			InoutFindByIDDeployment: new(model.Deployment),
			InputModelDeploymentStats: model.Stats{
				model.DeviceDeploymentStatusPending:     2,
				model.DeviceDeploymentStatusSuccess:     4,
				model.DeviceDeploymentStatusFailure:     1,
				model.DeviceDeploymentStatusInstalling:  3,
				model.DeviceDeploymentStatusRebooting:   3,
				model.DeviceDeploymentStatusDownloading: 3,
				model.DeviceDeploymentStatusAlreadyInst: 0,
			},

			OutputStats: model.Stats{
				model.DeviceDeploymentStatusDownloading: 3,
				model.DeviceDeploymentStatusRebooting:   3,
				model.DeviceDeploymentStatusInstalling:  3,
				model.DeviceDeploymentStatusSuccess:     4,
				model.DeviceDeploymentStatusFailure:     1,
				model.DeviceDeploymentStatusPending:     2,
				model.DeviceDeploymentStatusAlreadyInst: 0,
			},
		},
	}

	for testCaseNumber, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %d", testCaseNumber+1), func(t *testing.T) {
			t.Logf("testing %s %v %v", testCase.InputDeploymentID,
				testCase.InputModelDeploymentStats, testCase.InputModelError)

			db := new(mocks.DataStore)
			db.On("AggregateDeviceDeploymentByStatus",
				h.ContextMatcher(),
				testCase.InputDeploymentID).
				Return(testCase.InputModelDeploymentStats, testCase.InputModelError)

			db.On("FindDeploymentByID",
				h.ContextMatcher(),
				testCase.InputDeploymentID).
				Return(testCase.InoutFindByIDDeployment, testCase.InoutFindByIDError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DataStore: db,
			})

			stats, err := model.GetDeploymentStats(context.Background(),
				testCase.InputDeploymentID)

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
	//t.Parallel()

	statuses := []model.DeviceDeployment{}

	// common device status list for all tests
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
		statuses = append(statuses, *newdd)
	}

	testCases := map[string]struct {
		inDeploymentId string

		devsStorageStatuses []model.DeviceDeployment
		devsStorageErr      error

		depsStorageDeployment *model.Deployment
		depsStorageErr        error

		modelErr error
	}{
		"existing deployment with statuses": {
			inDeploymentId: "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",

			devsStorageStatuses: statuses,
			devsStorageErr:      nil,

			depsStorageDeployment: &model.Deployment{},
			depsStorageErr:        nil,

			modelErr: nil,
		},
		"deployment doesn't exist": {
			devsStorageStatuses: statuses,
			devsStorageErr:      nil,

			depsStorageDeployment: nil,
			depsStorageErr:        nil,

			modelErr: controller.ErrModelDeploymentNotFound,
		},
		"DeviceDeployments storage layer error": {
			inDeploymentId: "30b3e62c-9ec2-4312-a7fa-cff24cc7397b",

			devsStorageStatuses: nil,
			devsStorageErr:      errors.New("some verbose, low-level db error"),

			depsStorageDeployment: &model.Deployment{},
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

			db := new(mocks.DataStore)

			db.On("GetDeviceStatusesForDeployment",
				h.ContextMatcher(), tc.inDeploymentId).
				Return(tc.devsStorageStatuses, tc.devsStorageErr)

			db.On("FindDeploymentByID", h.ContextMatcher(), tc.inDeploymentId).
				Return(tc.depsStorageDeployment, tc.depsStorageErr)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DataStore: db,
			})
			statuses, err := model.GetDeviceStatusesForDeployment(context.Background(),
				tc.inDeploymentId)

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

	//t.Parallel()

	tref := time.Now()
	messages := []model.LogMessage{
		{
			Timestamp: &tref,
			Message:   "foo",
			Level:     "notice",
		},
	}
	testCases := []struct {
		InputDeploymentID string
		InputDeviceID     string
		InputLog          []model.LogMessage

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
			InputLog:           []model.LogMessage{},
			InputModelError:    nil,
			InputHasDeployment: true,

			OutputError: errors.New("Invalid deployment log: DeploymentID: ID:234 does not validate as uuidv4;messages: non zero value required"),
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

			db := new(mocks.DataStore)
			db.On("SaveDeviceDeploymentLog",
				h.ContextMatcher(),
				model.DeploymentLog{
					DeviceID:     testCase.InputDeviceID,
					DeploymentID: testCase.InputDeploymentID,
					Messages:     testCase.InputLog,
				}).
				Return(testCase.InputModelError)

			db.On("HasDeploymentForDevice",
				h.ContextMatcher(),
				testCase.InputDeploymentID, testCase.InputDeviceID).
				Return(testCase.InputHasDeployment, testCase.InputHasModelError)
			db.On("UpdateDeviceDeploymentLogAvailability",
				h.ContextMatcher(),
				testCase.InputDeviceID, testCase.InputDeploymentID, true).
				Return(testCase.InputUpdateLogError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DataStore: db,
			})

			err := model.SaveDeviceDeploymentLog(context.Background(),
				testCase.InputDeviceID,
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

	//t.Parallel()

	testCases := map[string]struct {
		MockDeployments []*model.Deployment
		MockError       error

		OutputError       error
		OutputDeployments []*model.Deployment
	}{
		"nothing found": {
			MockDeployments:   nil,
			OutputDeployments: []*model.Deployment{},
		},
		"error": {
			MockError:   errors.New("bad bad bad"),
			OutputError: errors.New("searching for deployments: bad bad bad"),
		},
		"found deployments": {
			MockDeployments:   []*model.Deployment{{Id: StringToPointer("lala")}},
			OutputDeployments: []*model.Deployment{{Id: StringToPointer("lala")}},
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			db := new(mocks.DataStore)
			db.On("Find",
				h.ContextMatcher(), mock.AnythingOfType("model.Query")).
				Return(testCase.MockDeployments, testCase.MockError)

			db.On("DeviceCountByDeployment",
				h.ContextMatcher(), mock.AnythingOfType("string")).
				Return(0, nil)

			dmodel := NewDeploymentModel(DeploymentsModelConfig{DataStore: db})

			deployments, err := dmodel.LookupDeployment(context.Background(),
				model.Query{})
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
	//t.Parallel()

	testCases := map[string]struct {
		InputDeploymentID string
		MockDeployment    *model.Deployment
		MockError         error

		OutputValue bool
		OutputError error
	}{
		"nothing found": {
			InputDeploymentID: "f826484e-1157-4109-af21-304e6d711561",
			MockDeployment:    nil,
			OutputError:       nil,
			OutputValue:       true,
		},
		"error": {
			InputDeploymentID: "f826484e-1157-4109-af21-304e6d711561",
			MockDeployment:    nil,
			MockError:         errors.New("bad bad bad"),
			OutputError:       errors.New("Searching for unfinished deployment by ID: bad bad bad"),
		},
		"found unfinished deployment": {
			InputDeploymentID: "f826484e-1157-4109-af21-304e6d711561",
			MockDeployment:    &model.Deployment{Id: StringToPointer("f826484e-1157-4109-af21-304e6d711561")},
			OutputValue:       false,
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			db := new(mocks.DataStore)
			db.On("FindUnfinishedByID",
				h.ContextMatcher(), mock.AnythingOfType("string")).
				Return(testCase.MockDeployment, testCase.MockError)

			model := NewDeploymentModel(DeploymentsModelConfig{DataStore: db})

			isFinished, err := model.IsDeploymentFinished(context.Background(),
				testCase.InputDeploymentID)
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
	//t.Parallel()

	testCases := map[string]struct {
		InputDeploymentID string

		AbortDeviceDeploymentsError            error
		AggregateDeviceDeploymentByStatusStats model.Stats
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
			AggregateDeviceDeploymentByStatusStats: model.Stats{},
			OutputError:                            errors.New("AggregateDeviceDeploymentByStatusError"),
		},
		"UpdateStatsAndFinishDeployment error": {
			InputDeploymentID:                      "f826484e-1157-4109-af21-304e6d711561",
			AggregateDeviceDeploymentByStatusStats: model.Stats{"aaa": 1},
			UpdateStatsAndFinishDeploymentError:    errors.New("UpdateStatsAndFinishDeploymentError"),
			OutputError:                            errors.New("UpdateStatsAndFinishDeploymentError"),
		},
		"all correct": {
			InputDeploymentID:                      "f826484e-1157-4109-af21-304e6d711561",
			AggregateDeviceDeploymentByStatusStats: model.Stats{"aaa": 1},
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			db := new(mocks.DataStore)
			db.On("AbortDeviceDeployments",
				h.ContextMatcher(), mock.AnythingOfType("string")).
				Return(testCase.AbortDeviceDeploymentsError)
			db.On("AggregateDeviceDeploymentByStatus",
				h.ContextMatcher(), mock.AnythingOfType("string")).
				Return(testCase.AggregateDeviceDeploymentByStatusStats,
					testCase.AggregateDeviceDeploymentByStatusError)
			db.On("UpdateStatsAndFinishDeployment",
				h.ContextMatcher(), mock.AnythingOfType("string"),
				mock.AnythingOfType("model.Stats")).
				Return(testCase.UpdateStatsAndFinishDeploymentError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DataStore: db,
			})

			err := model.AbortDeployment(context.Background(),
				testCase.InputDeploymentID)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeploymentModelDecommissionDevice(t *testing.T) {
	//t.Parallel()

	deviceDeployment, err := model.NewDeviceDeployment("foo", "bar")
	assert.NoError(t, err)

	testCases := map[string]struct {
		InputDeviceId string

		DecommissionDeviceDeploymentsError                   error
		FindAllDeploymentsForDeviceIDWithStatusesDeployments []model.DeviceDeployment
		FindAllDeploymentsForDeviceIDWithStatusesError       error
		AggregateDeviceDeploymentByStatusStats               model.Stats
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
			FindAllDeploymentsForDeviceIDWithStatusesDeployments: []model.DeviceDeployment{*deviceDeployment},
			AggregateDeviceDeploymentByStatusError:               errors.New("AggregateDeviceDeploymentByStatusError"),
			AggregateDeviceDeploymentByStatusStats:               model.Stats{},
			OutputError:                                          errors.New("AggregateDeviceDeploymentByStatusError"),
		},
		"UpdateStatsAndFinishDeployment error": {
			InputDeviceId: "foo",
			FindAllDeploymentsForDeviceIDWithStatusesDeployments: []model.DeviceDeployment{*deviceDeployment},
			AggregateDeviceDeploymentByStatusStats:               model.Stats{"aaa": 1},
			UpdateStatsAndFinishDeploymentError:                  errors.New("UpdateStatsAndFinishDeploymentError"),
			OutputError:                                          errors.New("UpdateStatsAndFinishDeploymentError"),
		},
		"all correct": {
			InputDeviceId: "foo",
			FindAllDeploymentsForDeviceIDWithStatusesDeployments: []model.DeviceDeployment{*deviceDeployment},
			AggregateDeviceDeploymentByStatusStats:               model.Stats{"aaa": 1},
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {

			db := new(mocks.DataStore)
			db.On("DecommissionDeviceDeployments",
				h.ContextMatcher(), mock.AnythingOfType("string")).
				Return(testCase.DecommissionDeviceDeploymentsError)
			db.On("FindAllDeploymentsForDeviceIDWithStatuses",
				h.ContextMatcher(),
				mock.AnythingOfType("string"),
				mock.AnythingOfType("[]string")).
				Return(testCase.FindAllDeploymentsForDeviceIDWithStatusesDeployments,
					testCase.FindAllDeploymentsForDeviceIDWithStatusesError)
			db.On("AggregateDeviceDeploymentByStatus",
				h.ContextMatcher(), mock.AnythingOfType("string")).
				Return(testCase.AggregateDeviceDeploymentByStatusStats,
					testCase.AggregateDeviceDeploymentByStatusError)
			db.On("UpdateStatsAndFinishDeployment",
				h.ContextMatcher(), mock.AnythingOfType("string"),
				mock.AnythingOfType("model.Stats")).
				Return(testCase.UpdateStatsAndFinishDeploymentError)

			model := NewDeploymentModel(DeploymentsModelConfig{
				DataStore: db,
			})

			err := model.DecommissionDevice(context.Background(),
				testCase.InputDeviceId)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
