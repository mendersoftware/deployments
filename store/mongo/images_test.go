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

package mongo

import (
	"context"
	"testing"

	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendersoftware/deployments/model"
)

func TestSoftwareImagesStorageImageByNameAndDeviceType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestDeploymentStorageImageByNameAndDeviceType in short mode.")
	}

	//image dataset - common for all cases
	inputImgs := bson.A{
		&model.SoftwareImage{
			Id: "1",
			SoftwareImageMetaConstructor: model.SoftwareImageMetaConstructor{
				Description: "description",
			},

			SoftwareImageMetaArtifactConstructor: model.SoftwareImageMetaArtifactConstructor{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"foo"},
				Updates:               []model.Update{},
			},
		},
		&model.SoftwareImage{
			Id: "2",
			SoftwareImageMetaConstructor: model.SoftwareImageMetaConstructor{
				Description: "description",
			},

			SoftwareImageMetaArtifactConstructor: model.SoftwareImageMetaArtifactConstructor{
				Name:                  "App2 v0.1",
				DeviceTypesCompatible: []string{"bar", "baz"},
				Updates:               []model.Update{},
			},
		},
	}

	//setup db - common for all cases
	ctx := context.Background()
	db.Wipe()
	client := db.Client()

	collection := client.Database(DatabaseName).Collection(CollectionImages)
	res, err := collection.InsertMany(ctx, inputImgs)
	assert.NoError(t, err)
	assert.Equal(t, len(res.InsertedIDs), len(inputImgs))

	testCases := map[string]struct {
		InputImageName string
		InputDevType   string
		InputTenant    string

		OutputImage *model.SoftwareImage
		OutputError error
	}{
		"name and dev type ok - single type": {
			InputImageName: "App1 v1.0",
			InputDevType:   "foo",

			OutputImage: inputImgs[0].(*model.SoftwareImage),
			OutputError: nil,
		},
		"name and dev type ok - multiple types": {
			InputImageName: "App2 v0.1",
			InputDevType:   "bar",

			OutputImage: inputImgs[1].(*model.SoftwareImage),
			OutputError: nil,
		},
		"name ok, dev type incompatible - single type": {
			InputImageName: "App1 v1.0",
			InputDevType:   "baz",

			OutputImage: nil,
			OutputError: nil,
		},
		"name ok, dev type incompatible - multiple types": {
			InputImageName: "App2 v0.1",
			InputDevType:   "foo",

			OutputImage: nil,
			OutputError: nil,
		},
		"name not found, dev type not found": {
			InputImageName: "App3 v0.1",
			InputDevType:   "bah",

			OutputImage: nil,
			OutputError: nil,
		},
		"name validation error": {
			InputImageName: "",
			InputDevType:   "foo",

			OutputImage: nil,
			OutputError: ErrSoftwareImagesStorageInvalidName,
		},
		"dev type validation error": {
			InputImageName: "App2 v0.1",
			InputDevType:   "",

			OutputImage: nil,
			OutputError: ErrSoftwareImagesStorageInvalidDeviceType,
		},
		"other tenant": {
			InputImageName: "App1 v1.0",
			InputDevType:   "foo",
			InputTenant:    "acme",

			OutputImage: nil,
			OutputError: nil,
		},
	}

	for name, tc := range testCases {

		// Run each test case as subtest
		t.Run(name, func(t *testing.T) {

			if tc.InputTenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Tenant: tc.InputTenant,
				})
			} else {
				ctx = context.Background()
			}
			store := NewDataStoreMongoWithClient(client)
			img, err := store.ImageByNameAndDeviceType(ctx,
				tc.InputImageName, tc.InputDevType)

			if tc.OutputError != nil {
				assert.EqualError(t, err, tc.OutputError.Error())
			} else {
				assert.NoError(t, err)

				if tc.OutputImage == nil {
					assert.Nil(t, img)
				} else {
					assert.NotNil(t, img)
					assert.Equal(t, *tc.OutputImage, *img)
				}
			}
		})
	}
}

func TestIsArtifactUnique(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestIsArtifactUnique in short mode.")
	}

	//image dataset - common for all cases
	inputImgs := []interface{}{
		&model.SoftwareImage{
			Id: "1",
			SoftwareImageMetaConstructor: model.SoftwareImageMetaConstructor{
				Description: "description",
			},

			SoftwareImageMetaArtifactConstructor: model.SoftwareImageMetaArtifactConstructor{
				Name:                  "app1-v1.0",
				DeviceTypesCompatible: []string{"foo", "bar"},
				Updates:               []model.Update{},
			},
		},
	}

	//setup db - common for all cases
	ctx := context.Background()
	db.Wipe()
	client := db.Client()

	collection := client.Database(DatabaseName).Collection(CollectionImages)
	_, err := collection.InsertMany(ctx, inputImgs)
	assert.NoError(t, err)

	testCases := map[string]struct {
		InputArtifactName string
		InputDevTypes     []string
		InputTenant       string

		OutputIsUnique bool
		OutputError    error
	}{
		"artifact unique - unique name": {
			InputArtifactName: "app1-v2.0",
			InputDevTypes:     []string{"foo", "bar"},

			OutputIsUnique: true,
			OutputError:    nil,
		},
		"artifact unique - unique platform": {
			InputArtifactName: "app1-v1.0",
			InputDevTypes:     []string{"baz"},

			OutputIsUnique: true,
			OutputError:    nil,
		},
		"artifact not unique": {
			InputArtifactName: "app1-v1.0",
			InputDevTypes:     []string{"foo", "baz"},

			OutputIsUnique: false,
			OutputError:    nil,
		},
		"empty artifact name": {
			InputDevTypes: []string{"baz", "bah"},

			OutputError: ErrSoftwareImagesStorageInvalidArtifactName,
		},
		"other tenant": {
			// is unique because we're using another DB
			InputArtifactName: "app1-v1.0",
			InputDevTypes:     []string{"foo", "bar"},
			InputTenant:       "acme",

			OutputIsUnique: true,
		},
	}

	for name, tc := range testCases {

		// Run test cases as subtests
		t.Run(name, func(t *testing.T) {

			if tc.InputTenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Tenant: tc.InputTenant,
				})
			} else {
				ctx = context.Background()
			}
			store := NewDataStoreMongoWithClient(client)
			isUnique, err := store.IsArtifactUnique(ctx,
				tc.InputArtifactName, tc.InputDevTypes)

			if tc.OutputError != nil {
				assert.EqualError(t, err, tc.OutputError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.OutputIsUnique, isUnique)
			}
		})
	}

}
