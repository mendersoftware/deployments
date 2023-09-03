// Copyright 2023 Northern.tech AS
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
	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	mstore "github.com/mendersoftware/go-lib-micro/store"
)

func TestMigration_1_2_19(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMigration_1_2_19 in short mode.")
	}

	db.Wipe()
	c := db.Client()

	ctx := context.TODO()

	//store := NewDataStoreMongoWithClient(c)
	database := c.Database(mstore.DbFromContext(ctx, DatabaseName))
	collRel := database.Collection(CollectionReleases)

	artifactType := "app"
	inputImgs := []*model.Image{
		{
			Id: "6d4f6e27-c3bb-438c-ad9c-d9de30e59d80",
			ImageMeta: &model.ImageMeta{
				Description: "description",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App1 v1.0",
				DeviceTypesCompatible: []string{"foo"},
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
		{
			Id: "6d4f6e27-c3bb-438c-ad9c-d9de30e59d85",
			ImageMeta: &model.ImageMeta{
				Description: "description2",
			},

			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "App4 v2.0",
				DeviceTypesCompatible: []string{"foo2"},
				Updates: []model.Update{
					{
						TypeInfo: model.ArtifactUpdateTypeInfo{
							Type: &artifactType,
						},
					},
				},
			},
			Modified: timePtr("2023-09-22T22:00:00+00:00"),
		},
	}
	releases := []model.Release{
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
		{
			Name: "App4 v2.0",
			Artifacts: []model.Image{
				*inputImgs[5],
			},
		},
	}
	items := make([]interface{}, len(releases))
	for i, _ := range releases {
		items[i] = releases[i]
	}
	r, err := collRel.InsertMany(ctx, items)
	assert.NotNil(t, r)
	assert.NoError(t, err)

	// apply migration (1.2.19)
	mnew := &migration_1_2_19{
		client: c,
		db:     DbName,
	}
	err = mnew.Up(migrate.MakeVersion(1, 2, 19))
	assert.NoError(t, err)

	indices := collRel.Indexes()
	exists, err := hasIndex(ctx, IndexNameReleaseArtifactsCount, indices)
	assert.NoError(t, err)
	assert.True(t, exists, "index "+IndexNameAggregatedUpdateTypes+" must exist in 1.2.19")

	cursor, err := collRel.Find(ctx, bson.M{})
	var releases010219 []model.Release
	err = cursor.All(ctx, &releases010219)
	assert.NoError(t, err)
	for _, r := range releases010219 {
		assert.Equal(t, len(r.Artifacts), r.ArtifactsCount)
	}
}
