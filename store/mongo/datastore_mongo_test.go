// Copyright 2022 Northern.tech AS
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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/go-lib-micro/identity"
	ctxstore "github.com/mendersoftware/go-lib-micro/store"
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

func timePtr(timeStr string) *time.Time {
	t, _ := time.Parse(time.RFC3339, timeStr)
	t = t.UTC()
	return &t
}

func TestGetReleases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestGetReleases in short mode.")
	}

	inputImgs := []*model.Image{
		{
			Id: "6d4f6e27-c3bb-438c-ad9c-d9de30e59d80",
			ImageMeta: &model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"foo"},
				Updates:               []model.Update{},
			},
			Modified: timePtr("2010-09-22T22:00:00+00:00"),
		},
		{
			Id: "6d4f6e27-c3bb-438c-ad9c-d9de30e59d81",
			ImageMeta: &model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App2 v0.1",
				DeviceTypesCompatible: []string{"foo"},
				Updates:               []model.Update{},
			},
			Modified: timePtr("2010-09-22T23:02:00+00:00"),
		},
		{
			Id: "6d4f6e27-c3bb-438c-ad9c-d9de30e59d82",
			ImageMeta: &model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"bar, baz"},
				Updates:               []model.Update{},
			},
			Modified: timePtr("2010-09-22T22:00:01+00:00"),
		},
		{
			Id: "6d4f6e27-c3bb-438c-ad9c-d9de30e59d83",
			ImageMeta: &model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"bork"},
				Updates:               []model.Update{},
			},
			Modified: timePtr("2010-09-22T22:00:04+00:00"),
		},
		{
			Id: "6d4f6e27-c3bb-438c-ad9c-d9de30e59d84",
			ImageMeta: &model.ImageMeta{
				Description: "extended description",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App2 v0.1",
				DeviceTypesCompatible: []string{"bar", "baz"},
				Updates:               []model.Update{},
			},
			Modified: timePtr("2010-09-22T23:00:00+00:00"),
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
		img.ArtifactMeta.Depends = make(map[string]interface{})
		img.ArtifactMeta.Depends["device_type"] = make(bson.A,
			len(img.ArtifactMeta.DeviceTypesCompatible),
		)
		for i, devType := range img.ArtifactMeta.DeviceTypesCompatible {
			img.ArtifactMeta.Depends["device_type"].(bson.A)[i] = devType
		}
	}

	testCases := map[string]struct {
		releaseFilt *model.ReleaseOrImageFilter

		releases []model.Release
		err      error
	}{
		"ok, all": {
			releases: []model.Release{
				{
					Name: "App1 v1.0",
					Artifacts: []model.Image{
						*inputImgs[0],
						*inputImgs[2],
						*inputImgs[3],
					},
				},
				{
					Name: "App2 v0.1",
					Artifacts: []model.Image{
						*inputImgs[1],
						*inputImgs[4],
					},
				},
			},
		},
		"ok, description partial": {
			releaseFilt: &model.ReleaseOrImageFilter{
				Description: "description",
			},
			releases: []model.Release{
				{
					Name: "App1 v1.0",
					Artifacts: []model.Image{
						*inputImgs[0],
						*inputImgs[2],
						*inputImgs[3],
					},
				},
				{
					Name: "App2 v0.1",
					Artifacts: []model.Image{
						*inputImgs[1],
						*inputImgs[4],
					},
				},
			},
		},
		"ok, description exact": {
			releaseFilt: &model.ReleaseOrImageFilter{
				Description: "extended description",
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
		"ok, sort by modified asc": {
			releaseFilt: &model.ReleaseOrImageFilter{
				Sort: "modified:asc",
			},
			releases: []model.Release{
				{
					Name: "App1 v1.0",
					Artifacts: []model.Image{
						*inputImgs[0],
						*inputImgs[2],
						*inputImgs[3],
					},
				},
				{
					Name: "App2 v0.1",
					Artifacts: []model.Image{
						*inputImgs[1],
						*inputImgs[4],
					},
				},
			},
		},
		"ok, sort by modified desc": {
			releaseFilt: &model.ReleaseOrImageFilter{
				Sort: "modified:desc",
			},
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
		"ok, device type": {
			releaseFilt: &model.ReleaseOrImageFilter{
				DeviceType: "bork",
			},
			releases: []model.Release{
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
		"ok, with sort and pagination": {
			releaseFilt: &model.ReleaseOrImageFilter{
				Sort:    "name:desc",
				Page:    2,
				PerPage: 1,
			},
			releases: []model.Release{
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
			releaseFilt: &model.ReleaseOrImageFilter{
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
			releaseFilt: &model.ReleaseOrImageFilter{
				Name: "App3 v1.0",
			},
			releases: []model.Release{},
		},
	}

	for name, tc := range testCases {

		t.Run(name, func(t *testing.T) {
			releases, count, err := ds.GetReleases(ctx, tc.releaseFilt)

			if tc.err != nil {
				assert.EqualError(t, tc.err, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.releases, releases)
			assert.GreaterOrEqual(t, count, len(tc.releases))
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
						Name:         "NYC Production",
						ArtifactName: "App 123",
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id:      "a108ae14-bb4e-455f-9b40-2ef4bab97bb7",
					Created: &now,
				},
				&model.Deployment{
					DeploymentConstructor: &model.DeploymentConstructor{
						Name:         "NYC Production",
						ArtifactName: "App 123",
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id:      "d1804903-5caa-4a73-a3ae-0efcc3205405",
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
						Name:         "NYC Production",
						ArtifactName: "App 123",
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id:      "a108ae14-bb4e-455f-9b40-2ef4bab97bb7",
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
						Name:         "NYC Production",
						ArtifactName: "App 123",
					},
					Id:     "a108ae14-bb4e-455f-9b40-2ef4bab97bb7",
					Active: true,
				},
			},
		},
		"one older deployment and one newer deployments": {
			InputDeploymentsCollection: []interface{}{
				&model.Deployment{
					DeploymentConstructor: &model.DeploymentConstructor{
						Name:         "NYC Production",
						ArtifactName: "App 123",
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id:      "a108ae14-bb4e-455f-9b40-2ef4bab97bb7",
					Created: TimePtr(now.Add(-time.Hour * 24)),
				},
				&model.Deployment{
					DeploymentConstructor: &model.DeploymentConstructor{
						Name:         "NYC Production",
						ArtifactName: "App 123",
						Devices:      []string{"b532b01a-9313-404f-8d19-e7fcbe5cc347"},
					},
					Id:      "d1804903-5caa-4a73-a3ae-0efcc3205405",
					Active:  true,
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
						Name:         "NYC Production",
						ArtifactName: "App 123",
					},
					Id:     "d1804903-5caa-4a73-a3ae-0efcc3205405",
					Active: true,
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
				Id:      "d50eda0d-2cea-4de1-8d42-9cd3e7e86701",
				Created: &now,
				DeploymentConstructor: &model.DeploymentConstructor{
					Name:         "name",
					ArtifactName: "artifact",
					Devices:      []string{"device-1"},
				},
			},
			count:    10,
			expected: 10,
		},
		"device_count is zero": {
			deployment: &model.Deployment{
				Id:          "d50eda0d-2cea-4de1-8d42-9cd3e7e86702",
				DeviceCount: &zero,
				Created:     &now,
				DeploymentConstructor: &model.DeploymentConstructor{
					Name:         "name",
					ArtifactName: "artifact",
					Devices:      []string{"device-1"},
				},
			},
			count:    10,
			expected: zero,
		},
		"device_count is one": {
			deployment: &model.Deployment{
				Id:          "d50eda0d-2cea-4de1-8d42-9cd3e7e86703",
				DeviceCount: &one,
				Created:     &now,
				DeploymentConstructor: &model.DeploymentConstructor{
					Name:         "name",
					ArtifactName: "artifact",
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

			err = ds.SetDeploymentDeviceCount(ctx, tc.deployment.Id, tc.count)
			assert.Nil(t, err)

			deployment, err := ds.FindDeploymentByID(ctx, tc.deployment.Id)
			assert.Nil(t, err)
			assert.NotNil(t, deployment)
			assert.NotNil(t, deployment.DeviceCount)
			assert.Equal(t, *deployment.DeviceCount, tc.expected)
		})
	}

}

func TestSetStorageSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestSetStorageSettings in short mode.")
	}

	testCases := map[string]struct {
		tenantID string
		settings *model.StorageSettings
		err      error
	}{
		"ok": {
			settings: &model.StorageSettings{
				Region: "region",
				Key:    "secretkey",
				Secret: "secret",
				Bucket: "bucket",
				Uri:    "https://example.com",
				Token:  "token",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			ds := NewDataStoreMongoWithClient(db.Client())

			err := ds.SetStorageSettings(ctx, tc.settings)
			assert.NoError(t, err)

			settings, err := ds.GetStorageSettings(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tc.settings, settings)
		})
	}
}

func TestSortDeployments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestSortDeployments in short mode.")
	}

	// Make sure we start test with empty database
	db.Wipe()

	uuidVal, _ := uuid.NewRandom()
	deploymentOneID := uuidVal.String()
	uuidVal, _ = uuid.NewRandom()
	deploymentTwoID := uuidVal.String()
	now := time.Now()
	startDate := now.AddDate(0, -1, 0)
	deviceCount := 1
	uuidVal, _ = uuid.NewRandom()
	devicesList := []string{uuidVal.String()}
	config := make([]byte, 0)
	inputDeployments := []*model.Deployment{
		{
			DeploymentConstructor: &model.DeploymentConstructor{
				Name:         "deployment 1",
				ArtifactName: "artifact 1",
			},
			Created:       &now,
			Id:            deploymentOneID,
			DeviceCount:   &deviceCount,
			MaxDevices:    1,
			DeviceList:    devicesList,
			Status:        model.DeploymentStatusInProgress,
			Type:          model.DeploymentTypeConfiguration,
			Configuration: config,
		},
		{
			DeploymentConstructor: &model.DeploymentConstructor{
				Name:         "deployment 2",
				ArtifactName: "artifact 2",
			},
			Created:       &startDate,
			Id:            deploymentTwoID,
			DeviceCount:   &deviceCount,
			MaxDevices:    1,
			DeviceList:    devicesList,
			Status:        model.DeploymentStatusFinished,
			Type:          model.DeploymentTypeConfiguration,
			Configuration: config,
		},
	}

	ctx := context.Background()
	ds := NewDataStoreMongoWithClient(db.Client())
	var deploymentsQty int64 = 2

	for _, depl := range inputDeployments {
		err := ds.InsertDeployment(ctx, depl)
		assert.NoError(t, err)
	}

	query := model.Query{
		Sort: model.SortDirectionDescending,
	}
	deployments, count, err := ds.Find(ctx, query)
	assert.NoError(t, err)
	assert.NotEmpty(t, deployments)
	assert.Equal(t, deploymentsQty, count)
	assert.Equal(t, deploymentOneID, deployments[0].Id)

	query = model.Query{
		Sort: model.SortDirectionAscending,
	}
	deployments, count, err = ds.Find(ctx, query)
	assert.NoError(t, err)
	assert.NotEmpty(t, deployments)
	assert.Equal(t, deploymentsQty, count)
	assert.Equal(t, deploymentTwoID, deployments[0].Id)
}

func TestFindOldestActiveDeviceDeployment(t *testing.T) {
	db.Wipe()
	const (
		TenantID       = "123456789012345678901234"
		TenantDeviceID = "27d5d258-b880-4157-8eb7-8d68aeb1663d"
		DeviceID       = "1140bc78-b898-4b2a-a4a2-551cb7bd9ac8"
	)
	// Initialize dataset
	ds := NewDataStoreMongoWithClient(db.Client())
	now := time.Now()
	for _, env := range []struct {
		tenantID string
		deviceID string
	}{
		{tenantID: TenantID, deviceID: TenantDeviceID},
		{deviceID: DeviceID},
	} {
		ctx := context.Background()
		if env.tenantID != "" {
			ctx = identity.WithContext(ctx, &identity.Identity{
				Tenant: env.tenantID,
			})
		}
		if err := ds.ProvisionTenant(ctx, env.tenantID); err != nil {
			panic(err)
		}
		for _, depl := range []*model.DeviceDeployment{{
			Id: "0",
			Created: func() *time.Time {
				ret := now.Add(-2 * time.Hour)
				return &ret
			}(),
			Status:       model.DeviceDeploymentStatusSuccess,
			DeviceId:     env.deviceID,
			DeploymentId: uuid.New().String(),
		}, {
			Id: "1",
			Created: func() *time.Time {
				ret := now.Add(-3 * time.Hour / 2)
				return &ret
			}(),
			Status:       model.DeviceDeploymentStatusAlreadyInst,
			DeviceId:     env.deviceID,
			DeploymentId: uuid.New().String(),
		}, {
			Id: "2",
			Created: func() *time.Time {
				ret := now.Add(-time.Hour)
				return &ret
			}(),
			Status:       model.DeviceDeploymentStatusDownloading,
			DeviceId:     env.deviceID,
			DeploymentId: uuid.New().String(),
		}, {
			Id: "3",
			Created: func() *time.Time {
				ret := now.Add(-time.Minute)
				return &ret
			}(),
			Status:       model.DeviceDeploymentStatusPending,
			DeviceId:     env.deviceID,
			DeploymentId: uuid.New().String(),
		}} {
			if err := ds.InsertDeviceDeployment(ctx, depl); err != nil {
				panic(err)
			}
		}
	}

	testCases := []struct {
		Name string

		CTX      context.Context
		TenantID string
		DeviceID string

		ExpectedID *string
		Error      error
	}{{
		Name: "ok",

		CTX:      context.Background(),
		DeviceID: DeviceID,

		ExpectedID: func() *string { s := "2"; return &s }(),
	}, {
		Name: "ok, multi-tenant mode",

		CTX: identity.WithContext(context.Background(), &identity.Identity{
			Tenant: TenantID,
		}),
		DeviceID: TenantDeviceID,

		ExpectedID: func() *string { s := "2"; return &s }(),
	}, {
		Name: "ok, no document",

		CTX: identity.WithContext(context.Background(), &identity.Identity{
			Tenant: TenantID,
		}),
		DeviceID: DeviceID, // NOTE: We're using the tenant database
	}, {
		Name: "error, context canceled",

		CTX: func() context.Context {
			ctx, ccl := context.WithCancel(context.Background())
			ccl()
			return ctx
		}(),
		DeviceID: DeviceID, // NOTE: We're using the tenant database

		Error: context.Canceled,
	}, {
		Name:  "error, empty device id",
		Error: ErrStorageInvalidID,
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			d, err := ds.FindOldestActiveDeviceDeployment(tc.CTX, tc.DeviceID)
			if tc.Error != nil {
				if assert.Error(t, err) {
					assert.Regexp(t, tc.Error.Error(), err.Error())
				}
			} else {
				if assert.NoError(t, err) {
					if tc.ExpectedID != nil && assert.NotNil(t, d) {
						assert.Equal(t, *tc.ExpectedID, d.Id,
							"did not receive the expected device deployment id",
						)
					} else {
						assert.Nil(t, d, "did not expect to find a deployment")
					}
				}
			}
		})
	}
}

func TestFindLatestInactiveDeviceDeployment(t *testing.T) {
	db.Wipe()
	const (
		TenantID       = "123456789012345678901234"
		TenantDeviceID = "27d5d258-b880-4157-8eb7-8d68aeb1663d"
		DeviceID       = "1140bc78-b898-4b2a-a4a2-551cb7bd9ac8"
	)
	// Initialize dataset
	ds := NewDataStoreMongoWithClient(db.Client())
	now := time.Now()
	for _, env := range []struct {
		tenantID string
		deviceID string
	}{
		{tenantID: TenantID, deviceID: TenantDeviceID},
		{deviceID: DeviceID},
	} {
		ctx := context.Background()
		if env.tenantID != "" {
			ctx = identity.WithContext(ctx, &identity.Identity{
				Tenant: env.tenantID,
			})
		}
		if err := ds.ProvisionTenant(ctx, env.tenantID); err != nil {
			panic(err)
		}
		for _, depl := range []*model.DeviceDeployment{{
			Id: "0",
			Created: func() *time.Time {
				ret := now.Add(-2 * time.Hour)
				return &ret
			}(),
			Status:       model.DeviceDeploymentStatusSuccess,
			DeviceId:     env.deviceID,
			DeploymentId: uuid.New().String(),
		}, {
			Id: "1",
			Created: func() *time.Time {
				ret := now.Add(-3 * time.Hour / 2)
				return &ret
			}(),
			Status:       model.DeviceDeploymentStatusAlreadyInst,
			DeviceId:     env.deviceID,
			DeploymentId: uuid.New().String(),
		}, {
			Id: "2",
			Created: func() *time.Time {
				ret := now.Add(-time.Hour)
				return &ret
			}(),
			Status:       model.DeviceDeploymentStatusDownloading,
			DeviceId:     env.deviceID,
			DeploymentId: uuid.New().String(),
		}, {
			Id: "3",
			Created: func() *time.Time {
				ret := now.Add(-time.Minute)
				return &ret
			}(),
			Status:       model.DeviceDeploymentStatusPending,
			DeviceId:     env.deviceID,
			DeploymentId: uuid.New().String(),
		}} {
			if err := ds.InsertDeviceDeployment(ctx, depl); err != nil {
				panic(err)
			}
		}
	}

	testCases := []struct {
		Name string

		CTX      context.Context
		TenantID string
		DeviceID string

		ExpectedID *string
		Error      error
	}{{
		Name: "ok",

		CTX:      context.Background(),
		DeviceID: DeviceID,

		ExpectedID: func() *string { s := "1"; return &s }(),
	}, {
		Name: "ok, multi-tenant mode",

		CTX: identity.WithContext(context.Background(), &identity.Identity{
			Tenant: TenantID,
		}),
		DeviceID: TenantDeviceID,

		ExpectedID: func() *string { s := "1"; return &s }(),
	}, {
		Name: "ok, no document",

		CTX: identity.WithContext(context.Background(), &identity.Identity{
			Tenant: TenantID,
		}),
		DeviceID: DeviceID, // NOTE: We're using the tenant database
	}, {
		Name: "error, context canceled",

		CTX: func() context.Context {
			ctx, ccl := context.WithCancel(context.Background())
			ccl()
			return ctx
		}(),
		DeviceID: DeviceID, // NOTE: We're using the tenant database

		Error: context.Canceled,
	}, {
		Name:  "error, empty device id",
		Error: ErrStorageInvalidID,
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			d, err := ds.FindLatestInactiveDeviceDeployment(tc.CTX, tc.DeviceID)
			if tc.Error != nil {
				if assert.Error(t, err) {
					assert.Regexp(t, tc.Error.Error(), err.Error())
				}
			} else {
				if assert.NoError(t, err) {
					if tc.ExpectedID != nil && assert.NotNil(t, d) {
						assert.Equal(t, *tc.ExpectedID, d.Id,
							"did not receive the expected device deployment id",
						)
					} else {
						assert.Nil(t, d, "did not expect to find a deployment")
					}
				}
			}
		})
	}
}
