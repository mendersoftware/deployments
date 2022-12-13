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

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	mstore "github.com/mendersoftware/go-lib-micro/store"
)

func TestMigration_1_2_13(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMigration_1_2_13 in short mode.")
	}

	testCases := map[string]struct {
		inputImage  *OldImage
		outputImage *model.Image
	}{
		"ok": {
			inputImage: &OldImage{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70f6",
				OldArtifactMeta: &OldArtifactMeta{
					Provides: map[string]string{"rootfs-image.checksum": "bar"},
				},
			},
			outputImage: &model.Image{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70f6",
				ArtifactMeta: &model.ArtifactMeta{
					Provides: model.Provides{"rootfs-image.checksum": "bar"},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Logf("test case: %s", name)

		db.Wipe()
		c := db.Client()

		ctx := context.TODO()

		//store := NewDataStoreMongoWithClient(c)
		database := c.Database(mstore.DbFromContext(ctx, DatabaseName))
		collImg := database.Collection(CollectionImages)

		// insert image
		_, err := collImg.InsertOne(ctx, tc.inputImage)
		assert.NoError(t, err)

		query := bson.M{
			model.StorageKeyImageProvidesKey:   "rootfs-image.checksum",
			model.StorageKeyImageProvidesValue: "bar",
		}
		// get old image using new query
		// there should be no documents in the result
		oldImage := OldImage{}
		err = collImg.FindOne(ctx, query).Decode(&oldImage)
		assert.EqualError(t, err, mongo.ErrNoDocuments.Error())

		// apply migration (1.2.13)
		mnew := &migration_1_2_13{
			client: c,
			db:     DbName,
		}
		err = mnew.Up(migrate.MakeVersion(1, 2, 13))
		assert.NoError(t, err)

		// get new image using new query
		// this time the image should be in the result
		image := model.Image{}
		err = collImg.FindOne(ctx, query).Decode(&image)
		assert.NoError(t, err)
		assert.Equal(t, *tc.outputImage, image)
	}

}
