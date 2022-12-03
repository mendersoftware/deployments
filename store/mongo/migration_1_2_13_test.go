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

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mendersoftware/deployments/model"
	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	mstore "github.com/mendersoftware/go-lib-micro/store"
)

type OldArtifactMeta struct {
	Name string `json:"name" bson:"name" valid:"length(1|4096),required"`
	//nolint:lll
	DeviceTypesCompatible []string               `json:"device_types_compatible" bson:"device_types_compatible" valid:"length(1|4096),required"`
	Info                  *model.ArtifactInfo    `json:"info"`
	Signed                bool                   `json:"signed" bson:"signed"`
	Updates               []model.Update         `json:"updates" valid:"-"`
	Provides              map[string]string      `json:"artifact_provides,omitempty" bson:"provides,omitempty" valid:"-"`
	Depends               map[string]interface{} `json:"artifact_depends,omitempty" bson:"depends" valid:"-"`
	//nolint:lll
	ClearsProvides []string `json:"clears_artifact_provides,omitempty" bson:"clears_provides,omitempty" valid:"-"`
}

type OldImage struct {
	// Image ID
	Id string `json:"id" bson:"_id" valid:"uuidv4,required"`

	// User provided field set
	*model.ImageMeta `bson:"meta"`

	// Field set provided with yocto image
	*OldArtifactMeta `bson:"meta_artifact"`

	// Artifact total size
	Size int64 `json:"size" bson:"size" valid:"-"`

	// Last modification time, including image upload time
	Modified *time.Time `json:"modified" valid:"-"`
}

func TestMigration_1_2_13(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMigration_1_2_13 in short mode.")
	}

	testCases := map[string]struct {
		inputImage  *OldImage
		outputImage *OldImage
	}{
		"ok": {
			inputImage: &OldImage{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70f6",
				OldArtifactMeta: &OldArtifactMeta{
					Provides: map[string]string{"rootfs-image.checksum": "bar"},
				},
			},
			outputImage: &OldImage{
				Id: "0cb87b3d-4f08-420b-b004-4347c07f70f6",
				OldArtifactMeta: &OldArtifactMeta{
					Provides: map[string]string{model.GetProvidesKeyReplacer().Replace("rootfs-image.checksum"): "bar"},
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
			model.StorageKeyImageProvidesRootFSChecksum: "bar",
		}
		// get image by key from provides
		// there should be no documents in the result
		image := OldImage{}
		err = collImg.FindOne(ctx, query).Decode(&image)
		assert.EqualError(t, err, mongo.ErrNoDocuments.Error())

		// apply migration (1.2.13)
		mnew := &migration_1_2_13{
			client: c,
			db:     DbName,
		}
		err = mnew.Up(migrate.MakeVersion(1, 2, 13))
		assert.NoError(t, err)

		// get image by key from provides
		// this time the image should be in the result
		err = collImg.FindOne(ctx, query).Decode(&image)
		assert.NoError(t, err)
		assert.Equal(t, *tc.outputImage, image)
	}

}
