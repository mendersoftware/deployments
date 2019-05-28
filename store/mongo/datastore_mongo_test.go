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

	"github.com/mendersoftware/deployments/model"
	images "github.com/mendersoftware/deployments/resources/images"
	mimages "github.com/mendersoftware/deployments/resources/images/mongo"
)

func TestGetReleases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestGetReleases in short mode.")
	}

	inputImgs := []interface{}{
		images.SoftwareImage{
			Id: "1",
			SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
				Description: "description",
			},

			SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"foo"},
				Updates:               []images.Update{},
			},
		},
		images.SoftwareImage{
			Id: "2",
			SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
				Description: "description",
			},

			SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
				Name:                  "App2 v0.1",
				DeviceTypesCompatible: []string{"foo"},
				Updates:               []images.Update{},
			},
		},
		&images.SoftwareImage{
			Id: "3",
			SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
				Description: "description",
			},

			SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"bar, baz"},
				Updates:               []images.Update{},
			},
		},
		&images.SoftwareImage{
			Id: "4",
			SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
				Description: "description",
			},

			SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"bork"},
				Updates:               []images.Update{},
			},
		},
		&images.SoftwareImage{
			Id: "5",
			SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
				Description: "description",
			},

			SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
				Name:                  "App2 v0.1",
				DeviceTypesCompatible: []string{"bar", "baz"},
				Updates:               []images.Update{},
			},
		},
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
					Artifacts: []images.SoftwareImage{
						images.SoftwareImage{
							Id: "2",
							SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
								Description: "description",
							},

							SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
								Name:                  "App2 v0.1",
								DeviceTypesCompatible: []string{"foo"},
								Updates:               []images.Update{},
							},
						},
						images.SoftwareImage{
							Id: "5",
							SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
								Description: "description",
							},

							SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
								Name:                  "App2 v0.1",
								DeviceTypesCompatible: []string{"bar", "baz"},
								Updates:               []images.Update{},
							},
						},
					},
				},
				model.Release{
					Name: "App1 v1.0",
					Artifacts: []images.SoftwareImage{
						images.SoftwareImage{
							Id: "1",
							SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
								Description: "description",
							},

							SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
								Name:                  "App1 v1.0",
								DeviceTypesCompatible: []string{"foo"},
								Updates:               []images.Update{},
							},
						},
						images.SoftwareImage{
							Id: "3",
							SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
								Description: "description",
							},

							SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
								Name:                  "App1 v1.0",
								DeviceTypesCompatible: []string{"bar, baz"},
								Updates:               []images.Update{},
							},
						},
						images.SoftwareImage{
							Id: "4",
							SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
								Description: "description",
							},

							SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
								Name:                  "App1 v1.0",
								DeviceTypesCompatible: []string{"bork"},
								Updates:               []images.Update{},
							},
						},
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
					Artifacts: []images.SoftwareImage{
						images.SoftwareImage{
							Id: "2",
							SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
								Description: "description",
							},

							SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
								Name:                  "App2 v0.1",
								DeviceTypesCompatible: []string{"foo"},
								Updates:               []images.Update{},
							},
						},
						images.SoftwareImage{
							Id: "5",
							SoftwareImageMetaConstructor: images.SoftwareImageMetaConstructor{
								Description: "description",
							},

							SoftwareImageMetaArtifactConstructor: images.SoftwareImageMetaArtifactConstructor{
								Name:                  "App2 v0.1",
								DeviceTypesCompatible: []string{"bar", "baz"},
								Updates:               []images.Update{},
							},
						},
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
			db.Wipe()

			s := NewDataStoreMongoWithSession(db.Session())
			defer s.session.Close()

			sess := s.session.Copy()
			defer sess.Close()

			coll := sess.DB(mimages.DatabaseName).C(mimages.CollectionImages)
			assert.NoError(t, coll.Insert(inputImgs...))

			releases, err := s.GetReleases(context.Background(), tc.releaseFilt)

			if tc.err != nil {
				assert.EqualError(t, tc.err, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.releases, releases)
		})
	}
}
