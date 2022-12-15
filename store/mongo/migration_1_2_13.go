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
	"fmt"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mendersoftware/deployments/model"
)

type migration_1_2_13 struct {
	client *mongo.Client
	db     string
}

type OldImage struct {
	// Image ID
	Id string `json:"id" bson:"_id" valid:"uuidv4,required"`

	// Field set provided with yocto image
	*OldArtifactMeta `bson:"meta_artifact"`
}

// Information provided by the Mender Artifact header
type OldArtifactMeta struct {
	// Provides is a map of artifact_provides used
	// for checking artifact (version 3) dependencies.
	//nolint:lll
	Provides map[string]string `json:"artifact_provides,omitempty" bson:"provides,omitempty" valid:"-"`
}

// Up intrduces index on artifact depends rootfs-image checksum and version and
// rewrites provides to avoid using dots and dollars in key names
func (m *migration_1_2_13) Up(from migrate.Version) error {
	ctx := context.Background()
	c := m.client.Database(m.db).Collection(CollectionImages)

	// transform existing device_types_compatible in v1 and v2 artifacts into 'depends.device_type'
	query := bson.M{
		StorageKeyImageProvides: bson.M{"$exists": true},
	}
	cursor, err := c.Find(ctx, query)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	var a OldImage
	for cursor.Next(ctx) {
		err = cursor.Decode(&a)
		if err != nil {
			break
		}

		provides := model.Provides(a.Provides)

		up := bson.M{
			"$set": bson.M{
				StorageKeyImageProvides: provides,
			},
		}

		res, err := c.UpdateOne(ctx, bson.M{"_id": a.Id}, up)
		if err != nil {
			return err
		}

		if res.MatchedCount != 1 {
			return errors.New(fmt.Sprintf("failed to update artifact %s: not found", a.Id))
		}
	}
	if err = cursor.Err(); err != nil {
		return errors.WithMessage(err, "failed to decode artifact")
	}

	// create new artifact provides rootfs checksum and version index
	storage := NewDataStoreMongoWithClient(m.client)
	err = storage.EnsureIndexes(m.db,
		CollectionImages,
		IndexArtifactProvides)

	return err
}

func (m *migration_1_2_13) Version() migrate.Version {
	return migrate.MakeVersion(1, 2, 13)
}
