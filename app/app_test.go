// Copyright 2021 Northern.tech AS
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

package app

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	inventory_mocks "github.com/mendersoftware/deployments/client/inventory/mocks"
	workflows_mocks "github.com/mendersoftware/deployments/client/workflows/mocks"
	"github.com/mendersoftware/deployments/model"
	fs_mocks "github.com/mendersoftware/deployments/s3/mocks"
	"github.com/mendersoftware/deployments/store/mocks"
	"github.com/mendersoftware/go-lib-micro/identity"
)

const (
	validUUIDv4  = "d50eda0d-2cea-4de1-8d42-9cd3e7e8670d"
	artifactSize = 10000
)

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name string

		DataStoreError error
		FileStoreError error
		WorkflowsError error
		InventoryError error
	}{{
		Name: "ok",
	}, {
		Name:           "error: datastore",
		DataStoreError: errors.New("connection error"),
	}, {
		Name:           "error: filestore",
		FileStoreError: errors.New("connection error"),
	}, {
		Name:           "error: workflows",
		WorkflowsError: errors.New("connection error"),
	}, {
		Name:           "error: inventory",
		InventoryError: errors.New("connection error"),
	}}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.TODO()
			mDStore := &mocks.DataStore{}
			mFStore := &fs_mocks.FileStorage{}
			mWorkflows := &workflows_mocks.Client{}
			mInventory := &inventory_mocks.Client{}
			dep := &Deployments{
				db:              mDStore,
				fileStorage:     mFStore,
				workflowsClient: mWorkflows,
				inventoryClient: mInventory,
			}
			switch {
			default:
				mInventory.On("CheckHealth", ctx).
					Return(tc.InventoryError)
				fallthrough
			case tc.WorkflowsError != nil:
				mWorkflows.On("CheckHealth", ctx).
					Return(tc.WorkflowsError)
				fallthrough
			case tc.FileStoreError != nil:
				mFStore.On("ListBuckets", ctx).
					Return(nil, tc.FileStoreError)
				fallthrough
			case tc.DataStoreError != nil:
				mDStore.On("Ping", ctx).
					Return(tc.DataStoreError)
			}
			err := dep.HealthCheck(ctx)
			switch {
			case tc.DataStoreError != nil:
				assert.EqualError(t, err,
					"error reaching MongoDB: "+
						tc.DataStoreError.Error(),
				)

			case tc.FileStoreError != nil:
				assert.EqualError(t, err,
					"error reaching artifact storage service: "+
						tc.FileStoreError.Error(),
				)

			case tc.WorkflowsError != nil:
				assert.EqualError(t, err,
					"Workflows service unhealthy: "+
						tc.WorkflowsError.Error(),
				)

			case tc.InventoryError != nil:
				assert.EqualError(t, err,
					"Inventory service unhealthy: "+
						tc.InventoryError.Error(),
				)
			default:
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeploymentModelCreateDeployment(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		InputConstructor *model.DeploymentConstructor

		InputDeploymentStorageInsertError error
		InputImagesByNameError            error

		InvDevices        []model.InvDevice
		InvDevicesPageTwo []model.InvDevice
		TotalCount        int
		SearchError       error
		GetFilterError    error

		OutputError error
		OutputBody  bool
	}{
		"model missing": {
			OutputError: ErrModelMissingInput,
		},
		"insert error": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "NYC Production",
				ArtifactName: "App 123",
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputDeploymentStorageInsertError: errors.New("insert error"),

			OutputError: errors.New("Storing deployment data: insert error"),
		},
		"ok": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "NYC Production",
				ArtifactName: "App 123",
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},

			OutputBody: true,
		},
		"ok with group": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "group",
				ArtifactName: "App 123",
				Group:        "group",
			},

			InvDevices: []model.InvDevice{
				{
					ID: "b532b01a-9313-404f-8d19-e7fcbe5cc347",
				},
			},
			TotalCount: 1,

			OutputBody: true,
		},
		"ok with group, two pages": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "group",
				ArtifactName: "App 123",
				Group:        "group",
			},

			InvDevices: []model.InvDevice{
				{
					ID: "b532b01a-9313-404f-8d19-e7fcbe5cc347",
				},
			},
			InvDevicesPageTwo: []model.InvDevice{
				{
					ID: "b532b01a-9313-404f-8d19-e7fcbe5cc348",
				},
			},
			TotalCount: 2,

			OutputBody: true,
		},
		"ko, with group, no device found": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "group",
				ArtifactName: "App 123",
				Group:        "group",
			},

			OutputError: ErrNoDevices,
		},
		"ko, with group, error while searching": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         "group",
				ArtifactName: "App 123",
				Group:        "group",
			},

			SearchError: errors.New("error searching inventory"),
			OutputError: ErrModelInternal,
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(fmt.Sprintf("test case %s", testCaseName), func(t *testing.T) {
			ctx := context.Background()

			identityObject := &identity.Identity{Tenant: "tenant_id"}
			ctx = identity.WithContext(ctx, identityObject)

			db := mocks.DataStore{}
			db.On("InsertDeployment",
				ctx,
				mock.AnythingOfType("*model.Deployment")).
				Return(testCase.InputDeploymentStorageInsertError)

			db.On("ImagesByName",
				ctx,
				mock.AnythingOfType("string")).
				Return(
					[]*model.Image{model.NewImage(
						validUUIDv4,
						&model.ImageMeta{},
						&model.ArtifactMeta{
							Name: "App 123",
							DeviceTypesCompatible: []string{
								"hammer",
							},
							Depends: map[string]interface{}{},
						}, artifactSize)},
					testCase.InputImagesByNameError)

			fs := &fs_mocks.FileStorage{}
			ds := NewDeployments(&db, fs, "")

			mockInventoryClient := &inventory_mocks.Client{}
			if testCase.InputConstructor != nil && testCase.InputConstructor.Group != "" && len(testCase.InputConstructor.Devices) == 0 {
				mockInventoryClient.On("Search", ctx,
					"tenant_id",
					model.SearchParams{
						Page:    1,
						PerPage: PerPageInventoryDevices,
						Filters: []model.FilterPredicate{
							{
								Scope:     InventoryIdentityScope,
								Attribute: InventoryStatusAttributeName,
								Type:      "$eq",
								Value:     InventoryStatusAccepted,
							},
							{
								Scope:     InventoryGroupScope,
								Attribute: InventoryGroupAttributeName,
								Type:      "$eq",
								Value:     testCase.InputConstructor.Group,
							},
						},
					},
				).Return(testCase.InvDevices, testCase.TotalCount, testCase.SearchError)

				if testCase.TotalCount > len(testCase.InvDevices) {
					mockInventoryClient.On("Search", ctx,
						"tenant_id",
						model.SearchParams{
							Page:    2,
							PerPage: PerPageInventoryDevices,
							Filters: []model.FilterPredicate{
								{
									Scope:     InventoryIdentityScope,
									Attribute: InventoryStatusAttributeName,
									Type:      "$eq",
									Value:     InventoryStatusAccepted,
								},
								{
									Scope:     InventoryGroupScope,
									Attribute: InventoryGroupAttributeName,
									Type:      "$eq",
									Value:     testCase.InputConstructor.Group,
								},
							},
						},
					).Return(testCase.InvDevicesPageTwo, testCase.TotalCount, testCase.SearchError)
				}
			}

			ds.SetInventoryClient(mockInventoryClient)

			out, err := ds.CreateDeployment(ctx, testCase.InputConstructor)
			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
			}
			if testCase.OutputBody {
				assert.NotNil(t, out)
			}

			mockInventoryClient.AssertExpectations(t)
		})
	}

}

func TestCreateDeviceConfigurationDeployment(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		inputConstructor  *model.ConfigurationDeploymentConstructor
		inputDeviceID     string
		inputDeploymentID string

		inputDeploymentStorageInsertError error

		outputError error
		outputID    string
	}{
		"ok": {
			inputConstructor: &model.ConfigurationDeploymentConstructor{
				Name:          "foo",
				Configuration: "bar",
			},
			inputDeviceID:     "foo-device",
			inputDeploymentID: "foo-deployment",

			outputID: "foo-deployment",
		},
		"constructor missing": {
			outputError: ErrModelMissingInput,
		},
		"insert error": {
			inputConstructor: &model.ConfigurationDeploymentConstructor{
				Name:          "foo",
				Configuration: "bar",
			},
			inputDeploymentStorageInsertError: errors.New("insert error"),

			outputError: errors.New("Storing deployment data: insert error"),
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("test case %s", name), func(t *testing.T) {
			ctx := context.Background()

			identityObject := &identity.Identity{Tenant: "tenant_id"}
			ctx = identity.WithContext(ctx, identityObject)

			db := mocks.DataStore{}
			db.On("InsertDeployment",
				ctx,
				mock.AnythingOfType("*model.Deployment")).
				Return(tc.inputDeploymentStorageInsertError)

			ds := &Deployments{
				db: &db,
			}

			out, err := ds.CreateDeviceConfigurationDeployment(ctx, tc.inputConstructor, tc.inputDeviceID, tc.inputDeploymentID)
			if tc.outputError != nil {
				assert.EqualError(t, err, tc.outputError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, out, tc.outputID)
			}
		})
	}
}
