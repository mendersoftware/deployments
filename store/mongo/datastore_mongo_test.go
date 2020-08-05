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

package mongo

import (
	"context"
	"testing"
	"time"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/go-lib-micro/identity"
	ctxstore "github.com/mendersoftware/go-lib-micro/store"

	"github.com/mendersoftware/deployments/utils/pointers"
)

func TestPing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestPing in short mode.")
	}
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancel()
	ds := NewDataStoreMongoWithClient(db.Client())
	err := ds.Ping(ctx)
	assert.NoError(t, err)
}

func TestGetReleases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestGetReleases in short mode.")
	}
	newID := func() string {
		return uuid.NewV4().String()
	}

	inputImgs := []*model.Image{
		{
			Id: newID(),
			ImageMeta: &model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"foo"},
				Updates:               []model.Update{},
			},
		},
		{
			Id: newID(),
			ImageMeta: &model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App2 v0.1",
				DeviceTypesCompatible: []string{"foo"},
				Updates:               []model.Update{},
			},
		},
		{
			Id: newID(),
			ImageMeta: &model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"bar, baz"},
				Updates:               []model.Update{},
			},
		},
		{
			Id: newID(),
			ImageMeta: &model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"bork"},
				Updates:               []model.Update{},
			},
		},
		{
			Id: newID(),
			ImageMeta: &model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App2 v0.1",
				DeviceTypesCompatible: []string{"bar", "baz"},
				Updates:               []model.Update{},
			},
		},
	}

	// Setup test context
	ctx := context.Background()
	ds := NewDataStoreMongoWithClient(db.Client())
	for _, img := range inputImgs {
		err := ds.InsertImage(ctx, img)
		assert.NoError(t, err)
		if err != nil {
			assert.FailNow(t,
				"error setting up image collection for testing")
		}

		// Convert Depends["device_type"] to bson.A for the sake of
		// simplifying test case definitions.
		img.Depends = make(map[string]interface{})
		img.Depends["device_type"] = make(bson.A, len(img.
			DeviceTypesCompatible))
		for i, devType := range img.DeviceTypesCompatible {
			img.Depends["device_type"].(bson.A)[i] = devType
		}
	}

	testCases := map[string]struct {
		releaseFilt *model.ReleaseFilter

		releases []model.Release
		err      error
	}{
		"ok, all": {
			releases: []model.Release{
				{
					Name: "App2 v0.1",
					Artifacts: []model.Image{
						*inputImgs[1],
						*inputImgs[4],
					},
				},
				{
					Name: "App1 v1.0",
					Artifacts: []model.Image{
						*inputImgs[0],
						*inputImgs[2],
						*inputImgs[3],
					},
				},
			},
		},
		"ok, by name": {
			releaseFilt: &model.ReleaseFilter{
				Name: "App2 v0.1",
			},
			releases: []model.Release{
				{
					Name: "App2 v0.1",
					Artifacts: []model.Image{
						*inputImgs[1],
						*inputImgs[4],
					},
				},
			},
		},
		"ok, not found": {
			releaseFilt: &model.ReleaseFilter{
				Name: "App3 v1.0",
			},
			releases: []model.Release{},
		},
	}

	for name, tc := range testCases {

		t.Run(name, func(t *testing.T) {
			releases, err := ds.GetReleases(ctx, tc.releaseFilt)

			if tc.err != nil {
				assert.EqualError(t, tc.err, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.releases, releases)
		})
	}
}

func TestFindNewerActiveDeployments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestFindNewerActiveDeployments in short mode.")
	}
	now := time.Now()

	testCases := map[string]struct {
		InputDeploymentsCollection []interface{}
		InputTenant                string
		InputCreatedAfter          *time.Time
		InputSkip                  int
		InputLimit                 int

		OutputError       error
		OutputDeployments []*model.Deployment
	}{
		"empty database": {
			InputCreatedAfter: &now,
			InputSkip:         0,
			InputLimit:        1,

			OutputError:       nil,
			OutputDeployments: nil,
		},
		"no newer deployments": {
			InputDeploymentsCollection: []interface{}{
				&model.Deployment{
					DeploymentConstructor: &model.DeploymentConstructor{
						Name:         pointers.StringToPointer("NYC Production"),
						ArtifactName: pointers.StringToPointer("App 123"),
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id:      pointers.StringToPointer("a108ae14-bb4e-455f-9b40-2ef4bab97bb7"),
					Created: &now,
				},
				&model.Deployment{
					DeploymentConstructor: &model.DeploymentConstructor{
						Name:         pointers.StringToPointer("NYC Production"),
						ArtifactName: pointers.StringToPointer("App 123"),
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id:      pointers.StringToPointer("d1804903-5caa-4a73-a3ae-0efcc3205405"),
					Created: &now,
				},
			},
			InputSkip:         0,
			InputLimit:        1,
			InputCreatedAfter: TimePtr(now.Add(time.Hour * 24)),

			OutputError:       nil,
			OutputDeployments: nil,
		},
		"one newer deployments": {
			InputDeploymentsCollection: []interface{}{
				&model.Deployment{
					DeploymentConstructor: &model.DeploymentConstructor{
						Name:         pointers.StringToPointer("NYC Production"),
						ArtifactName: pointers.StringToPointer("App 123"),
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id:      pointers.StringToPointer("a108ae14-bb4e-455f-9b40-2ef4bab97bb7"),
					Created: &now,
				},
			},
			InputSkip:         0,
			InputLimit:        5,
			InputCreatedAfter: TimePtr(now.Add(-time.Hour * 24)),

			OutputError: nil,
			OutputDeployments: []*model.Deployment{
				{
					DeploymentConstructor: &model.DeploymentConstructor{
						Name:         pointers.StringToPointer("NYC Production"),
						ArtifactName: pointers.StringToPointer("App 123"),
					},
					Id: pointers.StringToPointer("a108ae14-bb4e-455f-9b40-2ef4bab97bb7"),
				},
			},
		},
		"one older deployment and one newer deployments": {
			InputDeploymentsCollection: []interface{}{
				&model.Deployment{
					DeploymentConstructor: &model.DeploymentConstructor{
						Name:         pointers.StringToPointer("NYC Production"),
						ArtifactName: pointers.StringToPointer("App 123"),
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id:      pointers.StringToPointer("a108ae14-bb4e-455f-9b40-2ef4bab97bb7"),
					Created: TimePtr(now.Add(-time.Hour * 24)),
				},
				&model.Deployment{
					DeploymentConstructor: &model.DeploymentConstructor{
						Name:         pointers.StringToPointer("NYC Production"),
						ArtifactName: pointers.StringToPointer("App 123"),
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id:      pointers.StringToPointer("d1804903-5caa-4a73-a3ae-0efcc3205405"),
					Created: TimePtr(now.Add(time.Hour * 24)),
				},
			},
			InputSkip:         0,
			InputLimit:        5,
			InputCreatedAfter: &now,

			OutputError: nil,
			OutputDeployments: []*model.Deployment{
				{
					DeploymentConstructor: &model.DeploymentConstructor{
						Name:         pointers.StringToPointer("NYC Production"),
						ArtifactName: pointers.StringToPointer("App 123"),
					},
					Id: pointers.StringToPointer("d1804903-5caa-4a73-a3ae-0efcc3205405"),
				},
			},
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(testCaseName, func(t *testing.T) {

			// Make sure we start test with empty database
			db.Wipe()

			client := db.Client()
			store := NewDataStoreMongoWithClient(client)

			ctx := context.Background()
			if testCase.InputTenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Tenant: testCase.InputTenant,
				})
			} else {
				ctx = context.Background()
			}

			collDep := client.Database(ctxstore.
				DbFromContext(ctx, DatabaseName)).
				Collection(CollectionDeployments)

			if testCase.InputDeploymentsCollection != nil {
				_, err := collDep.InsertMany(
					ctx, testCase.InputDeploymentsCollection)
				assert.NoError(t, err)
			}

			deployments, err := store.FindNewerActiveDeployments(ctx,
				testCase.InputCreatedAfter, testCase.InputSkip, testCase.InputLimit)

			for i := range deployments {
				deployments[i].Created = nil
			}

			if testCase.OutputError != nil {
				assert.EqualError(t, err, testCase.OutputError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.OutputDeployments, deployments)
			}
		})
	}
}

func strToPtr(value string) *string {
	return &value
}

func TestSetDeploymentDeviceCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestSetDeploymentDeviceCount in short mode.")
	}

	zero := 0
	one := 1
	now := time.Now()

	testCases := map[string]struct {
		deployment *model.Deployment
		count      int
		expected   int
	}{
		"device_count doesn't exist": {
			deployment: &model.Deployment{
				Id:      strToPtr("d50eda0d-2cea-4de1-8d42-9cd3e7e86701"),
				Created: &now,
				DeploymentConstructor: &model.DeploymentConstructor{
					Name:         strToPtr("name"),
					ArtifactName: strToPtr("artifact"),
					Devices:      []string{"device-1"},
				},
			},
			count:    10,
			expected: 10,
		},
		"device_count is zero": {
			deployment: &model.Deployment{
				Id:          strToPtr("d50eda0d-2cea-4de1-8d42-9cd3e7e86702"),
				DeviceCount: &zero,
				Created:     &now,
				DeploymentConstructor: &model.DeploymentConstructor{
					Name:         strToPtr("name"),
					ArtifactName: strToPtr("artifact"),
					Devices:      []string{"device-1"},
				},
			},
			count:    10,
			expected: zero,
		},
		"device_count is one": {
			deployment: &model.Deployment{
				Id:          strToPtr("d50eda0d-2cea-4de1-8d42-9cd3e7e86703"),
				DeviceCount: &one,
				Created:     &now,
				DeploymentConstructor: &model.DeploymentConstructor{
					Name:         strToPtr("name"),
					ArtifactName: strToPtr("artifact"),
					Devices:      []string{"device-1"},
				},
			},
			count:    10,
			expected: one,
		},
	}

	ctx := context.Background()
	ds := NewDataStoreMongoWithClient(db.Client())

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := ds.InsertDeployment(ctx, tc.deployment)
			assert.Nil(t, err)

			err = ds.SetDeploymentDeviceCount(ctx, *tc.deployment.Id, tc.count)
			assert.Nil(t, err)

			deployment, err := ds.FindDeploymentByID(ctx, *tc.deployment.Id)
			assert.Nil(t, err)
			assert.NotNil(t, deployment)
			assert.NotNil(t, deployment.DeviceCount)
			assert.Equal(t, *deployment.DeviceCount, tc.expected)
		})
	}

}
