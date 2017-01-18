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

	"github.com/mendersoftware/deployments/resources/images"
	model "github.com/mendersoftware/deployments/resources/images/model"
	. "github.com/mendersoftware/deployments/resources/images/mongo"
	"github.com/stretchr/testify/assert"
)

func TestSoftwareImagesStorageImageByNameAndDeviceType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestDeploymentStorageImageByNameAndDeviceType in short mode.")
	}

	//image dataset - common for all cases
	inputImgs := []interface{}{
		&images.SoftwareImage{
			Id: "1",
			SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
				Description: "description",
			},

			SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
				Name: "App1 v1.0",
				DeviceTypesCompatible: []string{"foo"},
				Updates:               []images.Update{},
			},
		},
		&images.SoftwareImage{
			Id: "2",
			SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
				Description: "description",
			},

			SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
				Name: "App2 v0.1",
				DeviceTypesCompatible: []string{"bar", "baz"},
				Updates:               []images.Update{},
			},
		},
	}

	//setup db - common for all cases
	db.Wipe()
	session := db.Session()
	defer session.Close()

	coll := session.DB(DatabaseName).C(CollectionImages)
	assert.NoError(t, coll.Insert(inputImgs...))

	testCases := map[string]struct {
		InputImageName string
		InputDevType   string

		OutputImage *images.SoftwareImage
		OutputError error
	}{
		"name and dev type ok - single type": {
			InputImageName: "App1 v1.0",
			InputDevType:   "foo",

			OutputImage: inputImgs[0].(*images.SoftwareImage),
			OutputError: nil,
		},
		"name and dev type ok - multiple types": {
			InputImageName: "App2 v0.1",
			InputDevType:   "bar",

			OutputImage: inputImgs[1].(*images.SoftwareImage),
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
			OutputError: model.ErrSoftwareImagesStorageInvalidName,
		},
		"dev type validation error": {
			InputImageName: "App2 v0.1",
			InputDevType:   "",

			OutputImage: nil,
			OutputError: model.ErrSoftwareImagesStorageInvalidDeviceType,
		},
	}

	for name, tc := range testCases {

		// Run each test case as subtest
		t.Run(name, func(t *testing.T) {

			store := NewSoftwareImagesStorage(session)
			img, err := store.ImageByNameAndDeviceType(tc.InputImageName, tc.InputDevType)

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
		&images.SoftwareImage{
			Id: "1",
			SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
				Description: "description",
			},

			SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
				Name: "app1-v1.0",
				DeviceTypesCompatible: []string{"foo", "bar"},
				Updates:               []images.Update{},
			},
		},
	}

	//setup db - common for all cases
	db.Wipe()
	session := db.Session()
	defer session.Close()

	coll := session.DB(DatabaseName).C(CollectionImages)
	assert.NoError(t, coll.Insert(inputImgs...))

	testCases := map[string]struct {
		InputArtifactName string
		InputDevTypes     []string

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

			OutputError: model.ErrSoftwareImagesStorageInvalidArtifactName,
		},
	}

	for name, tc := range testCases {

		// Run test cases as subtests
		t.Run(name, func(t *testing.T) {

			store := NewSoftwareImagesStorage(session)
			isUnique, err := store.IsArtifactUnique(tc.InputArtifactName, tc.InputDevTypes)

			if tc.OutputError != nil {
				assert.EqualError(t, err, tc.OutputError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.OutputIsUnique, isUnique)
			}
		})
	}

}
