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

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendersoftware/deployments/model"
	dmodel "github.com/mendersoftware/deployments/model"
)

func TestGetReleases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestGetReleases in short mode.")
	}

	inputImgs := bson.A{
		model.Image{
			Id: "1",
			ImageMeta: model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: model.ArtifactMeta{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"foo"},
				Updates:               []model.Update{},
			},
		},
		model.Image{
			Id: "2",
			ImageMeta: model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: model.ArtifactMeta{
				Name:                  "App2 v0.1",
				DeviceTypesCompatible: []string{"foo"},
				Updates:               []model.Update{},
			},
		},
		&model.Image{
			Id: "3",
			ImageMeta: model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: model.ArtifactMeta{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"bar, baz"},
				Updates:               []model.Update{},
			},
		},
		&model.Image{
			Id: "4",
			ImageMeta: model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: model.ArtifactMeta{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"bork"},
				Updates:               []model.Update{},
			},
		},
		&model.Image{
			Id: "5",
			ImageMeta: model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: model.ArtifactMeta{
				Name:                  "App2 v0.1",
				DeviceTypesCompatible: []string{"bar", "baz"},
				Updates:               []model.Update{},
			},
		},
	}

	testCases := map[string]struct {
		releaseFilt *dmodel.ReleaseFilter

		releases []dmodel.Release
		err      error
	}{
		"ok, all": {
			releases: []dmodel.Release{
				dmodel.Release{
					Name: "App2 v0.1",
					Artifacts: []model.Image{
						model.Image{
							Id: "2",
							ImageMeta: model.ImageMeta{
								Description: "description",
							},

							ArtifactMeta: model.ArtifactMeta{
								Name:                  "App2 v0.1",
								DeviceTypesCompatible: []string{"foo"},
								Updates:               []model.Update{},
							},
						},
						model.Image{
							Id: "5",
							ImageMeta: model.ImageMeta{
								Description: "description",
							},

							ArtifactMeta: model.ArtifactMeta{
								Name:                  "App2 v0.1",
								DeviceTypesCompatible: []string{"bar", "baz"},
								Updates:               []model.Update{},
							},
						},
					},
				},
				dmodel.Release{
					Name: "App1 v1.0",
					Artifacts: []model.Image{
						model.Image{
							Id: "1",
							ImageMeta: model.ImageMeta{
								Description: "description",
							},

							ArtifactMeta: model.ArtifactMeta{
								Name:                  "App1 v1.0",
								DeviceTypesCompatible: []string{"foo"},
								Updates:               []model.Update{},
							},
						},
						model.Image{
							Id: "3",
							ImageMeta: model.ImageMeta{
								Description: "description",
							},

							ArtifactMeta: model.ArtifactMeta{
								Name:                  "App1 v1.0",
								DeviceTypesCompatible: []string{"bar, baz"},
								Updates:               []model.Update{},
							},
						},
						model.Image{
							Id: "4",
							ImageMeta: model.ImageMeta{
								Description: "description",
							},

							ArtifactMeta: model.ArtifactMeta{
								Name:                  "App1 v1.0",
								DeviceTypesCompatible: []string{"bork"},
								Updates:               []model.Update{},
							},
						},
					},
				},
			},
		},
		"ok, by name": {
			releaseFilt: &dmodel.ReleaseFilter{
				Name: "App2 v0.1",
			},
			releases: []dmodel.Release{
				dmodel.Release{
					Name: "App2 v0.1",
					Artifacts: []model.Image{
						model.Image{
							Id: "2",
							ImageMeta: model.ImageMeta{
								Description: "description",
							},

							ArtifactMeta: model.ArtifactMeta{
								Name:                  "App2 v0.1",
								DeviceTypesCompatible: []string{"foo"},
								Updates:               []model.Update{},
							},
						},
						model.Image{
							Id: "5",
							ImageMeta: model.ImageMeta{
								Description: "description",
							},

							ArtifactMeta: model.ArtifactMeta{
								Name:                  "App2 v0.1",
								DeviceTypesCompatible: []string{"bar", "baz"},
								Updates:               []model.Update{},
							},
						},
					},
				},
			},
		},
		"ok, not found": {
			releaseFilt: &dmodel.ReleaseFilter{
				Name: "App3 v1.0",
			},
			releases: []model.Release{},
		},
	}

	ctx := context.Background()
	for name, tc := range testCases {

		t.Run(name, func(t *testing.T) {
			db.Wipe()

			ds := NewDataStoreMongoWithClient(db.Client())

			collection := ds.client.Database(DatabaseName).
				Collection(CollectionImages)
			_, err := collection.InsertMany(ctx, inputImgs)
			assert.NoError(t, err)

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
