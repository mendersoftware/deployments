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

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendersoftware/deployments/model"
)

func TestGetReleases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestGetReleases in short mode.")
	}
	newID := func() string {
		return uuid.Must(uuid.NewV4()).String()
	}

	inputImgs := []*model.Image{
		&model.Image{
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
		&model.Image{
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
		&model.Image{
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
		&model.Image{
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
		&model.Image{
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
				model.Release{
					Name: "App2 v0.1",
					Artifacts: []model.Image{
						*inputImgs[1],
						*inputImgs[4],
					},
				},
				model.Release{
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
				model.Release{
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
