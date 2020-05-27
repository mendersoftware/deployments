// Copyright 2020 Northern.tech AS
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
	"github.com/mendersoftware/deployments/model"
	fs_mocks "github.com/mendersoftware/deployments/s3/mocks"
	"github.com/mendersoftware/deployments/store/mocks"
	"github.com/mendersoftware/deployments/utils/pointers"
	"github.com/mendersoftware/go-lib-micro/identity"
)

const (
	validUUIDv4  = "d50eda0d-2cea-4de1-8d42-9cd3e7e8670d"
	artifactSize = 10000
)

func TestDeploymentModelCreateDeployment(t *testing.T) {

	t.Parallel()

	testCases := map[string]struct {
		InputConstructor *model.DeploymentConstructor

		InputDeploymentStorageInsertError error
		InputImagesByNameError            error

		InvDevices     []model.InvDevice
		TotalCount     int
		SearchError    error
		GetFilterError error

		OutputError error
		OutputBody  bool
	}{
		"model missing": {
			OutputError: ErrModelMissingInput,
		},
		"insert error": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         pointers.StringToPointer("NYC Production"),
				ArtifactName: pointers.StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},
			InputDeploymentStorageInsertError: errors.New("insert error"),

			OutputError: errors.New("Storing deployment data: insert error"),
		},
		"ok": {
			InputConstructor: &model.DeploymentConstructor{
				Name:         pointers.StringToPointer("NYC Production"),
				ArtifactName: pointers.StringToPointer("App 123"),
				Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
			},

			OutputBody: true,
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
			mockInventoryClient.On("Search", ctx,
				"tenant_id",
				mock.AnythingOfType("model.SearchParams"),
			).Return(testCase.InvDevices, testCase.TotalCount, testCase.SearchError)

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
		})
	}

}
