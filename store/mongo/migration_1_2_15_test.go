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

func TestMigration_1_2_15(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMigration_1_2_15 in short mode.")
	}

	db.Wipe()
	c := db.Client()

	ctx := context.TODO()

	//store := NewDataStoreMongoWithClient(c)
	database := c.Database(mstore.DbFromContext(ctx, DatabaseName))
	collImg := database.Collection(CollectionImages)
	collRel := database.Collection(CollectionReleases)

	inputImages := []interface{}{
		&model.Image{
			Id: "0cb87b3d-4f08-420b-b004-4347c07f70f6",
			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "foo",
				DeviceTypesCompatible: []string{"foo"},
				Provides:              map[string]string{"rootfs-image.checksum": "bar"},
			},
		},
		&model.Image{
			Id: "0cb87b3d-4f08-420b-b004-4347c07f70f7",
			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "foo",
				DeviceTypesCompatible: []string{"foo"},
				Provides:              map[string]string{"rootfs-image.checksum": "bar"},
			},
		},
		&model.Image{
			Id: "0cb87b3d-4f08-420b-b004-4347c07f70f8",
			ArtifactMeta: &model.ArtifactMeta{
				Name:                  "bar",
				DeviceTypesCompatible: []string{"foo"},
				Provides:              map[string]string{"rootfs-image.checksum": "bar"},
			},
		},
	}

	outputReleases := []model.Release{
		{
			Name: "foo",
			Artifacts: []model.Image{
				{
					Id: "0cb87b3d-4f08-420b-b004-4347c07f70f6",
					ArtifactMeta: &model.ArtifactMeta{
						Name:                  "foo",
						DeviceTypesCompatible: []string{"foo"},
						Provides:              map[string]string{"rootfs-image.checksum": "bar"},
						Depends:               map[string]interface{}{"device_type": bson.A{"foo"}},
					},
				},
				{
					Id: "0cb87b3d-4f08-420b-b004-4347c07f70f7",
					ArtifactMeta: &model.ArtifactMeta{
						Name:                  "foo",
						DeviceTypesCompatible: []string{"foo"},
						Provides:              map[string]string{"rootfs-image.checksum": "bar"},
						Depends:               map[string]interface{}{"device_type": bson.A{"foo"}},
					},
				},
			},
		},
		{
			Name: "bar",
			Artifacts: []model.Image{
				{
					Id: "0cb87b3d-4f08-420b-b004-4347c07f70f8",
					ArtifactMeta: &model.ArtifactMeta{
						Name:                  "bar",
						DeviceTypesCompatible: []string{"foo"},
						Provides:              map[string]string{"rootfs-image.checksum": "bar"},
						Depends:               map[string]interface{}{"device_type": bson.A{"foo"}},
					},
				},
			},
		},
	}
	// insert images
	_, err := collImg.InsertMany(ctx, inputImages)
	assert.NoError(t, err)

	// get releases
	// there should be no documents in the result
	releases := []model.Release{}
	cursor, err := collRel.Find(ctx, bson.M{})
	assert.NoError(t, err)
	err = cursor.All(ctx, &releases)
	assert.NoError(t, err)
	assert.Len(t, releases, 0)

	// apply migration (1.2.15)
	mnew := &migration_1_2_15{
		client: c,
		db:     DbName,
	}
	err = mnew.Up(migrate.MakeVersion(1, 2, 15))
	assert.NoError(t, err)

	// get release
	// this time the releas should be in the result
	cursor, err = collRel.Find(ctx, bson.M{})
	assert.NoError(t, err)
	err = cursor.All(ctx, &releases)
	assert.NoError(t, err)
	assert.Len(t, releases, 2)
	// ignore modification timestamp
	for i := range releases {
		releases[i].Modified = nil
	}
	assert.Equal(t, outputReleases, releases)
}
